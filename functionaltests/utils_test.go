package functionaltests

import (
	"fmt"
	"os"
	"testing"
)

func ecAPICheck(t *testing.T) error {
	t.Helper()
	apiKey := os.Getenv("EC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("unable to obtain value from EC_API_KEY environment variable")
	}
	return nil
}
