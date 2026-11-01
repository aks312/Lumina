package metadata

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ExifInfo holds all extracted metadata from EXIF headers.
type ExifInfo struct {
	Make                string
	Model               string
	Software            string
	ImageDescription    string
	DateTimeOriginal    time.Time
	Orientation         int
	Width               int
	Height              int
	HasOpticalSensor    bool // ExposureTime, FNumber, ISO, or FocalLength detected
	ExposureTime        string
	FNumber             float64
	ISO                 int
	FocalLength         float64
	GPSLatitude         float64
	GPSLongitude        float64
	HasGPS              bool
	EmbeddedThumbOffset uint32
	EmbeddedThumbLength uint32
}

// ParseExifFromFile reads EXIF data from JPEG, TIFF, or RAW image files.
func ParseExifFromFile(filePath string) (*ExifInfo, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read first 128KB which covers all standard EXIF APP1 headers
	buf := make([]byte, 131072)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	data := buf[:n]

	return ParseExifFromBytes(data)
}

// ParseExifFromBytes inspects byte slices for JPEG APP1 or raw TIFF headers.
func ParseExifFromBytes(data []byte) (*ExifInfo, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("buffer too short for EXIF")
	}

	// 1. If JPEG, locate APP1 (0xFF 0xE1) marker
	if data[0] == 0xFF && data[1] == 0xD8 {
		return parseJpegApp1(data)
	}

	// 2. Direct TIFF header (II* or MM*)
	if (data[0] == 'I' && data[1] == 'I' && data[2] == 0x2A && data[3] == 0x00) ||
		(data[0] == 'M' && data[1] == 'M' && data[2] == 0x00 && data[3] == 0x2A) {
		return parseTiffPayload(data, 0)
	}

	// 3. Search for "Exif\x00\x00" within data (e.g. from HEIC or WebP chunks)
	exifIdx := bytes.Index(data, []byte("Exif\x00\x00"))
	if exifIdx != -1 && len(data) > exifIdx+6 {
		return parseTiffPayload(data[exifIdx+6:], 0)
	}

	return nil, fmt.Errorf("no EXIF header found")
}

func parseJpegApp1(data []byte) (*ExifInfo, error) {
	idx := 2
	for idx < len(data)-4 {
		if data[idx] != 0xFF {
			idx++
			continue
		}
		marker := data[idx+1]
		if marker == 0xDA || marker == 0xD9 { // Start of Scan or End of Image
			break
		}
		if idx+4 > len(data) {
			break
		}
		segLen := int(binary.BigEndian.Uint16(data[idx+2 : idx+4]))
		if segLen < 2 {
			break
		}

		if marker == 0xE1 { // APP1 segment
			payload := data[idx+4 : min(len(data), idx+2+segLen)]
			if len(payload) >= 6 && string(payload[:6]) == "Exif\x00\x00" {
				return parseTiffPayload(payload[6:], idx+10)
			}
		}

		idx += 2 + segLen
	}
	return nil, fmt.Errorf("no APP1 EXIF segment found in JPEG")
}

func parseTiffPayload(tiff []byte, fileOffsetBase int) (*ExifInfo, error) {
	if len(tiff) < 8 {
		return nil, fmt.Errorf("tiff data too short")
	}

	var byteOrder binary.ByteOrder
	if tiff[0] == 'I' && tiff[1] == 'I' {
		byteOrder = binary.LittleEndian
	} else if tiff[0] == 'M' && tiff[1] == 'M' {
		byteOrder = binary.BigEndian
	} else {
		return nil, fmt.Errorf("invalid tiff byte order")
	}

	if byteOrder.Uint16(tiff[2:4]) != 42 {
		return nil, fmt.Errorf("invalid tiff magic")
	}

	firstIFDOffset := int(byteOrder.Uint32(tiff[4:8]))
	if firstIFDOffset >= len(tiff) {
		return nil, fmt.Errorf("first IFD offset out of bounds")
	}

	info := &ExifInfo{}
	visited := make(map[int]bool)

	parseIFD(tiff, firstIFDOffset, byteOrder, info, visited, fileOffsetBase)

	return info, nil
}

func parseIFD(tiff []byte, ifdOffset int, order binary.ByteOrder, info *ExifInfo, visited map[int]bool, fileOffsetBase int) {
	if visited[ifdOffset] || ifdOffset+2 > len(tiff) {
		return
	}
	visited[ifdOffset] = true

	numEntries := int(order.Uint16(tiff[ifdOffset : ifdOffset+2]))
	curr := ifdOffset + 2

	var subExifOffset, gpsOffset int
	var thumbOffset, thumbLen uint32

	for i := 0; i < numEntries; i++ {
		if curr+12 > len(tiff) {
			break
		}
		tag := order.Uint16(tiff[curr : curr+2])
		typ := order.Uint16(tiff[curr+2 : curr+4])
		count := order.Uint32(tiff[curr+4 : curr+8])
		valBytes := tiff[curr+8 : curr+12]

		var valOffset int
		valSize := getTiffTypeSize(typ) * int(count)
		if valSize <= 4 {
			valOffset = curr + 8
		} else {
			valOffset = int(order.Uint32(valBytes))
		}

		switch tag {
		case 0x010F: // Make
			info.Make = readString(tiff, valOffset, count)
		case 0x0110: // Model
			info.Model = readString(tiff, valOffset, count)
		case 0x0131: // Software
			info.Software = readString(tiff, valOffset, count)
		case 0x010E: // ImageDescription
			info.ImageDescription = readString(tiff, valOffset, count)
		case 0x0112: // Orientation
			info.Orientation = int(order.Uint16(valBytes[:2]))
		case 0x0100: // ImageWidth
			if typ == 3 {
				info.Width = int(order.Uint16(valBytes[:2]))
			} else {
				info.Width = int(order.Uint32(valBytes))
			}
		case 0x0101: // ImageLength / Height
			if typ == 3 {
				info.Height = int(order.Uint16(valBytes[:2]))
			} else {
				info.Height = int(order.Uint32(valBytes))
			}
		case 0x8769: // SubIFD (ExifIFD)
			subExifOffset = int(order.Uint32(valBytes))
		case 0x8825: // GPSInfo IFD
			gpsOffset = int(order.Uint32(valBytes))
		case 0x0201: // JPEGInterchangeFormat (thumbnail offset)
			thumbOffset = order.Uint32(valBytes)
		case 0x0202: // JPEGInterchangeFormatLength (thumbnail length)
			thumbLen = order.Uint32(valBytes)

		// Optical Sensor proof tags (Rule 2 Screenshot Veto)
		case 0x829A: // ExposureTime
			info.HasOpticalSensor = true
			if valOffset+8 <= len(tiff) {
				num := order.Uint32(tiff[valOffset : valOffset+4])
				den := order.Uint32(tiff[valOffset+4 : valOffset+8])
				if den > 0 {
					info.ExposureTime = fmt.Sprintf("%d/%d", num, den)
				}
			}
		case 0x829D: // FNumber
			info.HasOpticalSensor = true
			if valOffset+8 <= len(tiff) {
				num := order.Uint32(tiff[valOffset : valOffset+4])
				den := order.Uint32(tiff[valOffset+4 : valOffset+8])
				if den > 0 {
					info.FNumber = float64(num) / float64(den)
				}
			}
		case 0x8827: // ISO Speed
			info.HasOpticalSensor = true
			info.ISO = int(order.Uint16(valBytes[:2]))
		case 0x920A: // FocalLength
			info.HasOpticalSensor = true
			if valOffset+8 <= len(tiff) {
				num := order.Uint32(tiff[valOffset : valOffset+4])
				den := order.Uint32(tiff[valOffset+4 : valOffset+8])
				if den > 0 {
					info.FocalLength = float64(num) / float64(den)
				}
			}
		case 0x9003, 0x9004, 0x0132: // DateTimeOriginal, DateTimeDigitized, DateTime
			if info.DateTimeOriginal.IsZero() {
				str := readString(tiff, valOffset, count)
				if parsed, err := parseExifTime(str); err == nil {
					info.DateTimeOriginal = parsed
				}
			}
		case 0xA002: // ExifImageWidth
			if info.Width == 0 {
				if typ == 3 {
					info.Width = int(order.Uint16(valBytes[:2]))
				} else {
					info.Width = int(order.Uint32(valBytes))
				}
			}
		case 0xA003: // ExifImageHeight
			if info.Height == 0 {
				if typ == 3 {
					info.Height = int(order.Uint16(valBytes[:2]))
				} else {
					info.Height = int(order.Uint32(valBytes))
				}
			}
		}

		curr += 12
	}

	if thumbOffset > 0 && thumbLen > 0 && int(thumbOffset+thumbLen) <= len(tiff) {
		info.EmbeddedThumbOffset = uint32(fileOffsetBase) + thumbOffset
		info.EmbeddedThumbLength = thumbLen
	}

	// Recurse into SubIFD if present
	if subExifOffset > 0 && subExifOffset < len(tiff) {
		parseIFD(tiff, subExifOffset, order, info, visited, fileOffsetBase)
	}

	// Recurse into GPS IFD if present
	if gpsOffset > 0 && gpsOffset < len(tiff) {
		parseGPSIFD(tiff, gpsOffset, order, info, visited)
	}

	// Check next IFD (IFD1 typically holds thumbnail)
	if curr+4 <= len(tiff) {
		nextIFD := int(order.Uint32(tiff[curr : curr+4]))
		if nextIFD > 0 && nextIFD < len(tiff) {
			parseIFD(tiff, nextIFD, order, info, visited, fileOffsetBase)
		}
	}
}

func parseGPSIFD(tiff []byte, offset int, order binary.ByteOrder, info *ExifInfo, visited map[int]bool) {
	if visited[offset] || offset+2 > len(tiff) {
		return
	}
	visited[offset] = true

	numEntries := int(order.Uint16(tiff[offset : offset+2]))
	curr := offset + 2

	var latRef, lonRef string
	var latCoords, lonCoords [3]float64

	for i := 0; i < numEntries; i++ {
		if curr+12 > len(tiff) {
			break
		}
		tag := order.Uint16(tiff[curr : curr+2])
		typ := order.Uint16(tiff[curr+2 : curr+4])
		count := order.Uint32(tiff[curr+4 : curr+8])
		valBytes := tiff[curr+8 : curr+12]

		var valOffset int
		valSize := getTiffTypeSize(typ) * int(count)
		if valSize <= 4 {
			valOffset = curr + 8
		} else {
			valOffset = int(order.Uint32(valBytes))
		}

		switch tag {
		case 0x0001: // GPSLatitudeRef
			latRef = readString(tiff, valOffset, count)
		case 0x0002: // GPSLatitude
			latCoords = readRationals3(tiff, valOffset, order)
		case 0x0003: // GPSLongitudeRef
			lonRef = readString(tiff, valOffset, count)
		case 0x0004: // GPSLongitude
			lonCoords = readRationals3(tiff, valOffset, order)
		}

		curr += 12
	}

	if latCoords[0] > 0 || latCoords[1] > 0 || latCoords[2] > 0 {
		deg := latCoords[0] + latCoords[1]/60.0 + latCoords[2]/3600.0
		if strings.ToUpper(latRef) == "S" {
			deg = -deg
		}
		info.GPSLatitude = deg
		info.HasGPS = true
	}
	if lonCoords[0] > 0 || lonCoords[1] > 0 || lonCoords[2] > 0 {
		deg := lonCoords[0] + lonCoords[1]/60.0 + lonCoords[2]/3600.0
		if strings.ToUpper(lonRef) == "W" {
			deg = -deg
		}
		info.GPSLongitude = deg
		info.HasGPS = true
	}
}

func readRationals3(tiff []byte, offset int, order binary.ByteOrder) [3]float64 {
	var res [3]float64
	for i := 0; i < 3; i++ {
		if offset+8 > len(tiff) {
			break
		}
		num := order.Uint32(tiff[offset : offset+4])
		den := order.Uint32(tiff[offset+4 : offset+8])
		if den > 0 {
			res[i] = float64(num) / float64(den)
		}
		offset += 8
	}
	return res
}

func getTiffTypeSize(typ uint16) int {
	switch typ {
	case 1, 2, 7: // BYTE, ASCII, UNDEFINED
		return 1
	case 3: // SHORT
		return 2
	case 4: // LONG
		return 4
	case 5: // RATIONAL
		return 8
	default:
		return 1
	}
}

func readString(tiff []byte, offset int, count uint32) string {
	if offset < 0 || offset >= len(tiff) {
		return ""
	}
	end := offset + int(count)
	if end > len(tiff) {
		end = len(tiff)
	}
	str := string(tiff[offset:end])
	return strings.TrimRight(str, "\x00 \r\n")
}

func parseExifTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	formats := []string{
		"2006:01:02 15:04:05",
		"2006-01-02 15:04:05",
		"2006:01:02T15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
	}
	for _, fmtStr := range formats {
		if t, err := time.Parse(fmtStr, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown exif time format: %s", s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
