package metadata

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

// HeicMetadata contains parsed width, height, and EXIF tags from a .heic file.
type HeicMetadata struct {
	Width            int
	Height           int
	Make             string
	Model            string
	DateTimeOriginal time.Time
	Orientation      int
	HasOpticalSensor bool
	ExposureTime     string
	FNumber          float64
	ISO              int
	FocalLength      float64
}

// ParseHeic parses an ISO BMFF HEIC / HEIF file directly without external tools.
func ParseHeic(filePath string) (*HeicMetadata, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read the first 256KB which contains ftyp, meta, iprp, ispe, and EXIF header
	buf := make([]byte, 262144)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	data := buf[:n]

	return ParseHeicFromBytes(data)
}

// ParseHeicFromBytes inspects BMFF boxes for 'ftyp', 'ispe', and EXIF TIFF.
func ParseHeicFromBytes(data []byte) (*HeicMetadata, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("buffer too short for HEIC")
	}

	// Verify 'ftyp'
	boxSize := binary.BigEndian.Uint32(data[0:4])
	if boxSize < 8 {
		return nil, fmt.Errorf("invalid box size")
	}
	boxType := string(data[4:8])
	if boxType != "ftyp" {
		return nil, fmt.Errorf("missing ftyp box")
	}

	res := &HeicMetadata{}

	// 1. Scan for 'ispe' (Image Spatial Extents) box
	// ispe payload: [version: 1 byte][flags: 3 bytes][width: 4 bytes BE][height: 4 bytes BE]
	ispeIdx := 0
	for {
		idx := bytes.Index(data[ispeIdx:], []byte("ispe"))
		if idx == -1 {
			break
		}
		absoluteIdx := ispeIdx + idx
		// box header starts 4 bytes before type if 32-bit size
		if absoluteIdx >= 4 && absoluteIdx+16 <= len(data) {
			width := int(binary.BigEndian.Uint32(data[absoluteIdx+8 : absoluteIdx+12]))
			height := int(binary.BigEndian.Uint32(data[absoluteIdx+12 : absoluteIdx+16]))
			if width > 0 && height > 0 {
				// Keep the largest dimensions found (primary item rather than thumbnail)
				if width*height > res.Width*res.Height {
					res.Width = width
					res.Height = height
				}
			}
		}
		ispeIdx = absoluteIdx + 4
	}

	// 2. Scan for EXIF TIFF header within the HEIC stream
	// Look for "Exif\x00\x00" or direct TIFF magic (II*\x00 or MM\x00*)
	exifIdx := bytes.Index(data, []byte("Exif\x00\x00"))
	var tiffPayload []byte

	if exifIdx != -1 && len(data) > exifIdx+6 {
		tiffPayload = data[exifIdx+6:]
	} else {
		// Scan for TIFF header: II (0x49492A00) or MM (0x4D4D002A)
		for i := 16; i < len(data)-8; i++ {
			if (data[i] == 'I' && data[i+1] == 'I' && data[i+2] == 0x2A && data[i+3] == 0x00) ||
				(data[i] == 'M' && data[i+1] == 'M' && data[i+2] == 0x00 && data[i+3] == 0x2A) {
				tiffPayload = data[i:]
				break
			}
		}
	}

	if len(tiffPayload) >= 8 {
		exifInfo, err := parseTiffPayload(tiffPayload, 0)
		if err == nil && exifInfo != nil {
			res.Make = exifInfo.Make
			res.Model = exifInfo.Model
			res.DateTimeOriginal = exifInfo.DateTimeOriginal
			res.Orientation = exifInfo.Orientation
			res.HasOpticalSensor = exifInfo.HasOpticalSensor
			res.ExposureTime = exifInfo.ExposureTime
			res.FNumber = exifInfo.FNumber
			res.ISO = exifInfo.ISO
			res.FocalLength = exifInfo.FocalLength

			if res.Width == 0 && exifInfo.Width > 0 {
				res.Width = exifInfo.Width
			}
			if res.Height == 0 && exifInfo.Height > 0 {
				res.Height = exifInfo.Height
			}
		}
	}

	return res, nil
}
