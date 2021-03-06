package commands

import (
	"fmt"

	"github.com/cloudfoundry/bosh-bootloader/flags"
	"github.com/cloudfoundry/bosh-bootloader/storage"
)

type DeleteLBs struct {
	gcpDeleteLBs   gcpDeleteLBs
	awsDeleteLBs   awsDeleteLBs
	logger         logger
	stateValidator stateValidator
}

type gcpDeleteLBs interface {
	Execute(state storage.State) error
}

type awsDeleteLBs interface {
	Execute(state storage.State) error
}

func NewDeleteLBs(gcpDeleteLBs gcpDeleteLBs, awsDeleteLBs awsDeleteLBs,
	logger logger, stateValidator stateValidator) DeleteLBs {
	return DeleteLBs{
		gcpDeleteLBs:   gcpDeleteLBs,
		awsDeleteLBs:   awsDeleteLBs,
		logger:         logger,
		stateValidator: stateValidator,
	}
}

func (d DeleteLBs) Execute(subcommandFlags []string, state storage.State) error {
	config, err := d.parseFlags(subcommandFlags)
	if err != nil {
		return err
	}

	if config.skipIfMissing && !lbExists(state.Stack.LBType) {
		d.logger.Println("no lb type exists, skipping...")
		return nil
	}

	err = d.stateValidator.Validate()
	if err != nil {
		return err
	}

	switch state.IAAS {
	case "gcp":
		return d.gcpDeleteLBs.Execute(state)
	case "aws":
		return d.awsDeleteLBs.Execute(state)
	default:
		return fmt.Errorf("%q is an invalid iaas type in state, supported iaas types are: [gcp, aws]", state.IAAS)
	}

	return nil
}

func (DeleteLBs) parseFlags(subcommandFlags []string) (deleteLBsConfig, error) {
	lbFlags := flags.New("delete-lbs")

	config := deleteLBsConfig{}
	lbFlags.Bool(&config.skipIfMissing, "skip-if-missing", "", false)

	err := lbFlags.Parse(subcommandFlags)
	if err != nil {
		return config, err
	}

	return config, nil
}
