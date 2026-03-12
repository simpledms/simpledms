package account

type PasskeyPolicy struct {
	IsAccountPasskeyEnabled bool
	IsTenantPasskeyEnforced bool
	HasPasskeyCredentials   bool
}

func NewPasskeyPolicy(
	isAccountPasskeyEnabled bool,
	isTenantPasskeyEnforced bool,
	hasPasskeyCredentials bool,
) *PasskeyPolicy {
	return &PasskeyPolicy{
		IsAccountPasskeyEnabled: isAccountPasskeyEnabled,
		IsTenantPasskeyEnforced: isTenantPasskeyEnforced,
		HasPasskeyCredentials:   hasPasskeyCredentials,
	}
}

func (qq *PasskeyPolicy) IsPasswordSignInAllowed() bool {
	if qq.IsAccountPasskeyEnabled {
		return false
	}

	if !qq.IsTenantPasskeyEnforced {
		return true
	}

	return !qq.HasPasskeyCredentials
}

func (qq *PasskeyPolicy) IsPasskeyLoginRequired() bool {
	return qq.IsAccountPasskeyEnabled || qq.IsTenantPasskeyEnforced
}

func (qq *PasskeyPolicy) IsTenantPasskeyEnrollmentRequired() bool {
	if qq.IsAccountPasskeyEnabled {
		return false
	}

	if !qq.IsTenantPasskeyEnforced {
		return false
	}

	return !qq.HasPasskeyCredentials
}
