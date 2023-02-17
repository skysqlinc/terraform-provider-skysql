package provider

import (
	"context"
	"github.com/asaskevich/govalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"net"
)

type allowListIPValidator struct{}

// Description returns a plain text description of the validator's behavior, suitable for a practitioner to understand its impact.
func (v allowListIPValidator) Description(ctx context.Context) string {
	return "IP address must be in a CIDR IP address format, that looks like a normal IP address except that it ends with a slash followed by a number, called the IP network prefix."
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior, suitable for a practitioner to understand its impact.
func (v allowListIPValidator) MarkdownDescription(ctx context.Context) string {
	return "IP address must be in a CIDR IP address format, that looks like a normal IP address except that it ends with a slash followed by a number, called the IP network prefix."
}

// ValidateString Validate runs the main validation logic of the validator, reading configuration data out of `req` and updating `resp` with diagnostics.
func (v allowListIPValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// If the value is unknown or null, there is nothing to validate.
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	ip := req.ConfigValue.ValueString()
	if !govalidator.IsCIDR(ip) && govalidator.IsIP(ip) {
		ip += "/32"
	}
	if !isValidCIDR(ip) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Incorrect IP address format",
			"IP address must be in a CIDR IP address format, that looks like a normal IP address except that it ends with a slash followed by a number, called the IP network prefix.",
		)
	}
}

func isValidCIDR(cidr string) bool {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		// input is not a valid CIDR notation
		return false
	}

	if cidr == ipnet.String() {
		return true
	}

	return false
}

// Contains checks if slice contains a value
func Contains[T comparable](slice []T, value T) bool {
	for _, a := range slice {
		if a == value {
			return true
		}
	}

	return false
}

func toPtr[t any](u t) *t { return &u }
