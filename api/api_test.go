package api

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func DefaultTestApiOpts() ApiOpts {
	return ApiOpts{
		port: ":8080",
	}
}

func Test_NewApiOpts(t *testing.T) {
	tests := []struct {
		name, port string
		want       ApiOpts
		wantErr    bool
	}{
		{
			name: "when all opts are given",
			port: ":8080",
			want: ApiOpts{
				port: ":8080",
			},
			wantErr: false,
		},
		{
			name:    "when port is nil",
			port:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		got, gotErr := NewApiOpts(tt.port)
		if (gotErr != nil) != tt.wantErr {
			t.Errorf("NewApiOpts() got error does not match want error. gotErr: %v, wantErr: %t", gotErr, tt.wantErr)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("NewApiOpts() got: %v, want: %v", got, tt.want)
		}
	}
}

func Test_NewApi(t *testing.T) {
	tests := []struct {
		name     string
		opts     ApiOpts
		wantErr  bool
		validate func(t *testing.T, got *Api)
	}{
		{
			name:    "when all options are given",
			wantErr: false,
			opts:    DefaultTestApiOpts(),
			validate: func(t *testing.T, got *Api) {
				if got == nil {
					t.Fatalf("Expected non-nil Api, got nil")
				}
				if got.server == nil || got.server.Addr != ":8080" {
					t.Errorf("Server Addr = %v, want %v", got.server.Addr, ":8080")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotApp, gotErr := NewApi(tt.opts)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("NewApp() error = %v, wantErr = %v", gotErr, tt.wantErr)
				return
			}
			if tt.validate != nil {
				tt.validate(t, gotApp)
			}
			if gotApp != nil {
				gotApp.Shutdown()
			}
		})
	}
}

func TestStartMultipleGoRoutines(t *testing.T) {
	opts := DefaultTestApiOpts()
	app, err := NewApi(opts)
	if err != nil {
		t.Errorf("StartMultipleGoRoutines() error initializing app: %v", err)
		return
	}
	numRoutines := 5
	executed := make(chan struct{}, numRoutines)
	routine := func(ctx context.Context) {
		executed <- struct{}{}
	}
	for i := 0; i < numRoutines; i++ {
		app.startGoRoutine(routine)
	}
	time.Sleep(10 * time.Millisecond)
	if len(executed) != numRoutines {
		t.Errorf("Expected %d routines to execute, got %d", numRoutines, len(executed))
	}
	app.wg.Wait()
}

func TestApp_StartHttpServer(t *testing.T) {
	opts := DefaultTestApiOpts()
	app, err := NewApi(opts)
	if err != nil {
		t.Errorf("StartHttpServer() error initializing app: %v", err)
		return
	}
	app.StartHttpServer()
	if app.server == nil {
		t.Errorf("app server is nil, expected to be initialized")
	}
	app.Shutdown()
}
