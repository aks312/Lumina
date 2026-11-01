package main

import (
	"embed"
	"fmt"
	"net/http"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:             "Lumina Photos V3",
		Width:             1366,
		Height:            860,
		MinWidth:          1024,
		MinHeight:         680,
		BackgroundColour:  &options.RGBA{R: 15, G: 23, B: 42, A: 240},
		OnStartup:         app.startup,
		OnShutdown:        app.shutdown,
		Bind: []any{
			app,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.URL.Path, "/thumbs/") || strings.HasPrefix(r.URL.Path, "/thumb/") {
						app.ServeThumbnail(w, r)
						return
					}
					if strings.HasPrefix(r.URL.Path, "/raw/") {
						app.ServeRawMedia(w, r)
						return
					}
					// CRITICAL: Pass all other requests (/, /index.html, /assets/*.js, /assets/*.css) to the embedded frontend!
					next.ServeHTTP(w, r)
				})
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
			BackdropType:                      windows.Acrylic, // Next-gen Acrylic translucency
			Theme:                             windows.Dark,
			DisableWindowIcon:                 false,
			IsZoomControlEnabled:              false,
			ZoomFactor:                        1.0,
			CustomTheme: &windows.ThemeSettings{
				DarkModeTitleBar:   windows.RGB(15, 23, 42),
				DarkModeTitleText:  windows.RGB(248, 250, 252),
				DarkModeBorder:     windows.RGB(51, 65, 85),
			},
		},
	})

	if err != nil {
		fmt.Printf("Wails runtime error: %v\n", err)
	}
}
