package main

import (
	"encoding/json"
	"net"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

func (app *App) runServer() {
	listen, err := net.Listen("tcp4", "127.0.0.1:8080")
	if err != nil {
		logger.Fatal("bind failed",
			zap.Error(err))
	}

	var contents []byte
	err = fasthttp.Serve(listen, func(ctx *fasthttp.RequestCtx) {
		contents, err = json.Marshal(app.getPackageList())
		if err != nil {
			logger.Error("failed to encode package list",
				zap.Error(err))
			ctx.Error("failed to encode", 500)
			return
		}

		ctx.SetContentType("application/json")
		_, err := ctx.Write(contents)
		if err != nil {
			logger.Fatal("failed to write",
				zap.Error(err))
		}
	})

	if err != nil {
		logger.Fatal("serve failed",
			zap.Error(err))
	}
}
