package cli

import (
	"errors"
	"reflect"
	"testing"

	"github.com/geodro/lerd/internal/serviceops"
)

func TestOutdatedServices_KeepsOnlyServicesWithAnUpdate(t *testing.T) {
	check := func(name string) (*serviceops.UpdateAvailability, error) {
		switch name {
		case "mailpit":
			return &serviceops.UpdateAvailability{Available: true, LatestTag: "v1.28"}, nil
		case "redis":
			return nil, errors.New("registry unreachable")
		}
		return &serviceops.UpdateAvailability{}, nil
	}
	got := outdatedServices([]string{"mysql", "mailpit", "redis"}, check)
	if want := []string{"mailpit"}; !reflect.DeepEqual(got, want) {
		t.Errorf("outdatedServices = %v, want %v", got, want)
	}
}

func TestServiceUpdateCmd_AllRefusesAServiceName(t *testing.T) {
	cmd := newServiceUpdateCmd()
	if cmd.Flags().Lookup("all") == nil {
		t.Fatal("--all flag missing on `service update`")
	}
	cmd.SetArgs([]string{"--all", "mysql"})
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	if err := cmd.Execute(); err == nil {
		t.Error("`service update --all mysql` should be refused")
	}
}

func TestServiceUpdateCmd_NeedsANameWithoutAll(t *testing.T) {
	cmd := newServiceUpdateCmd()
	cmd.SetArgs([]string{})
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	if err := cmd.Execute(); err == nil {
		t.Error("`service update` with no service and no --all should be refused")
	}
}
