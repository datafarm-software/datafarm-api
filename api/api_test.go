package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"reflect"
	"testing"
	"time"

	"gomodules.xyz/memfs"
)

type MockMetadataFetcher struct{}

func (m *MockMetadataFetcher) Close() error                                         { return nil }
func (m *MockMetadataFetcher) GetMapValue(deviceId, mapKey string) (string, error)  { return "", nil }
func (m *MockMetadataFetcher) GetAttachedSensors(deviceId string) ([]string, error) { return nil, nil }
func (m *MockMetadataFetcher) GetQueryFields(attachedSensors []string) ([]string, error) {
	return nil, nil
}

type MockDataFetcher struct{}

func (m *MockDataFetcher) GetData(metadata Metadata) ([]byte, error) {
	return nil, nil
}

func (m *MockDataFetcher) Close() error { return nil }

type MockAuthoriser struct{}

func (m *MockAuthoriser) GenerateJwt() (string, error) { return "", nil }

func DefaultTestApiOpts() ApiOpts {
	return ApiOpts{
		port:            ":8080",
		publicKeyFile:   "key.pub",
		fileSystem:      memfs.New(),
		metadataFetcher: &MockMetadataFetcher{},
		dataFetcher:     &MockDataFetcher{},
		authoriser:      &MockAuthoriser{},
	}
}

func GetFileSystemWithPublicKeyFile(t *testing.T, fileName string) *memfs.FS {
	fs := memfs.New()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Errorf("Error generating key: %v", err)
	}
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Errorf("Error marshalling public key: %v", err)
	}
	publicPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})
	if err := fs.WriteFile(fileName, publicPem, 0644); err != nil {
		t.Fatalf("Failed to create mock file %s: %v", fileName, err)
	}
	return fs
}

func Test_NewApiOpts(t *testing.T) {
	fs := GetFileSystemWithPublicKeyFile(t, "key.pub")
	tests := []struct {
		name, port, pkf string
		testFS          *memfs.FS
		mf              metadataFetcher
		df              dataFetcher
		au              authoriser
		want            ApiOpts
		wantErr         bool
	}{
		{
			name:   "when all opts are given",
			port:   ":8080",
			testFS: fs,
			pkf:    "key.pub",
			mf:     &MockMetadataFetcher{},
			df:     &MockDataFetcher{},
			au:     &MockAuthoriser{},
			want: ApiOpts{
				port:            ":8080",
				publicKeyFile:   "key.pub",
				fileSystem:      fs,
				metadataFetcher: &MockMetadataFetcher{},
				dataFetcher:     &MockDataFetcher{},
				authoriser:      &MockAuthoriser{},
			},
			wantErr: false,
		},
		{
			name:    "when no options are given",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		got, gotErr := NewApiOpts(tt.port, tt.pkf, tt.testFS, tt.mf, tt.df, tt.au)
		if (gotErr != nil) != tt.wantErr {
			t.Errorf("NewApiOpts() got error does not match want error. gotErr: %v, wantErr: %t", gotErr, tt.wantErr)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("NewApiOpts() got: %v, want: %v", got, tt.want)
		}
	}
}

func Test_NewApi(t *testing.T) {
	opts := DefaultTestApiOpts()
	opts.fileSystem = GetFileSystemWithPublicKeyFile(t, "key.pub")
	tests := []struct {
		name     string
		opts     ApiOpts
		wantErr  bool
		validate func(t *testing.T, got *Api)
	}{
		{
			name:    "when all options are given",
			wantErr: false,
			opts:    opts,
			validate: func(t *testing.T, got *Api) {
				if got == nil {
					t.Fatalf("Expected non-nil Api, got nil")
				}
				if got.server == nil || got.server.Addr != ":8080" {
					t.Errorf("Server Addr = %v, want %v", got.server.Addr, ":8080")
				}
				if got.metadataFetcher == nil {
					t.Errorf("Api metadatafetcher is not initialized")
				}
				if got.dataFetcher == nil {
					t.Errorf("Api datafetcher is not initialized")
				}
				if got.authoriser == nil {
					t.Errorf("Api authoriser is not initialized")
				}
				if got.publicKey == nil {
					t.Errorf("Api publicKey is not initialized")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotApi, gotErr := NewApi(tt.opts)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("NewApi() error = %v, wantErr = %v", gotErr, tt.wantErr)
				return
			}
			if tt.validate != nil {
				tt.validate(t, gotApi)
			}
			if gotApi != nil {
				gotApi.Shutdown()
			}
		})
	}
}

func TestStartMultipleGoRoutines(t *testing.T) {
	opts := DefaultTestApiOpts()
	opts.fileSystem = GetFileSystemWithPublicKeyFile(t, "key.pub")
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
	opts.fileSystem = GetFileSystemWithPublicKeyFile(t, "key.pub")
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

func TestApi_formatQueryRange(t *testing.T) {
	opts := DefaultTestApiOpts()
	opts.fileSystem = GetFileSystemWithPublicKeyFile(t, "key.pub")
	tests := []struct {
		name      string
		startTime string
		stopTime  string
		want      string
		wantErr   bool
	}{
		{
			name:      "relative time",
			startTime: "-6d",
			stopTime:  "can be anything",
			want:      "start: -6d",
			wantErr:   false,
		},
		{
			name:      "no start time provided",
			startTime: "",
			stopTime:  "can be anything",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "rfc3339 start time provided, no stop time",
			startTime: "2025-01-29T08:00:00Z",
			stopTime:  "",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "rfc3339 start and stop time provided",
			startTime: "2025-01-28T08:00:00Z",
			stopTime:  "2025-01-29T08:00:00Z",
			want:      "start: 2025-01-28T08:00:00Z, stop: 2025-01-29T08:00:00Z",
			wantErr:   false,
		},
		{
			name:      "relative start time with rfc3339 stop time",
			startTime: "-6d",
			stopTime:  "2025-01-29T08:00:00Z",
			want:      "start: -6d",
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := NewApi(opts)
			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}
			got, gotErr := a.formatQueryRange(tt.startTime, tt.stopTime)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("formatQueryRange() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("formatQueryRange() succeeded unexpectedly")
			}
			if got != tt.want {
				t.Errorf("formatQueryRange() = %s, want %s", got, tt.want)
			}
		})
	}
}
