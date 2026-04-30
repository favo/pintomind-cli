package appctx

import (
	"context"

	"favo/pintomind-cli/internal/api"
	"favo/pintomind-cli/internal/config"
)

type key struct{}

type App struct {
	Config           *config.Config
	Client           *api.Client
	ActiveConnection string
	JSONOutput       bool
	Verbose          bool
}

func WithApp(ctx context.Context, app *App) context.Context {
	return context.WithValue(ctx, key{}, app)
}

func FromContext(ctx context.Context) *App {
	return ctx.Value(key{}).(*App)
}
