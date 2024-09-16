package crdupgradesafety

import (
	kappcus "carvel.dev/kapp/pkg/kapp/crdupgradesafety"
	"context"
	"errors"
	"fmt"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
)

type Option func(p *CRDUpgradeChecker)

func WithValidator(v *kappcus.Validator) Option {
	return func(p *CRDUpgradeChecker) {
		p.validator = v
	}
}

type CRDUpgradeChecker struct {
	crdClient apiextensionsv1client.CustomResourceDefinitionInterface
	validator *kappcus.Validator
}

func NewCRDUpgradeChecker(crdCli apiextensionsv1client.CustomResourceDefinitionInterface, opts ...Option) *CRDUpgradeChecker {
	changeValidations := []kappcus.ChangeValidation{
		kappcus.EnumChangeValidation,
		kappcus.RequiredFieldChangeValidation,
		kappcus.MaximumChangeValidation,
		kappcus.MaximumItemsChangeValidation,
		kappcus.MaximumLengthChangeValidation,
		kappcus.MaximumPropertiesChangeValidation,
		kappcus.MinimumChangeValidation,
		kappcus.MinimumItemsChangeValidation,
		kappcus.MinimumLengthChangeValidation,
		kappcus.MinimumPropertiesChangeValidation,
		kappcus.DefaultValueChangeValidation,
	}
	p := &CRDUpgradeChecker{
		crdClient: crdCli,
		// create a default validator. Can be overridden via the options
		validator: &kappcus.Validator{
			Validations: []kappcus.Validation{
				kappcus.NewValidationFunc("NoScopeChange", kappcus.NoScopeChange),
				kappcus.NewValidationFunc("NoStoredVersionRemoved", kappcus.NoStoredVersionRemoved),
				kappcus.NewValidationFunc("NoExistingFieldRemoved", kappcus.NoExistingFieldRemoved),
				&ServedVersionValidator{Validations: changeValidations},
				&kappcus.ChangeValidator{Validations: changeValidations},
			},
		},
	}

	for _, o := range opts {
		o(p)
	}

	return p
}

func (p *CRDUpgradeChecker) Check(ctx context.Context, oldCrd *apiextensionsv1.CustomResourceDefinition, newCrd *apiextensionsv1.CustomResourceDefinition) error {
	var validateErrors []error

	err := p.validator.Validate(*oldCrd, *newCrd)
	if err != nil {
		validateErrors = append(validateErrors, fmt.Errorf("validating upgrade for CRD %q failed: %w", newCrd.Name, err))
	}

	return errors.Join(validateErrors...)
}
