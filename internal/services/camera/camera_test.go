package camera

import (
	"net/http"
	"testing"
)

func TestNewService(t *testing.T) {
	client := &http.Client{}
	service := NewService(client)

	if service == nil {
		t.Errorf("Expected NewService to return a non-nil service")
	}

	if service.Tasks == nil {
		t.Errorf("Expected service.Tasks to be non-nil")
	}

	if service.Config == nil {
		t.Errorf("Expected service.Config to be non-nil")
	}
}
