package choices

import "slices"

// Purpose is a synthetic, illustrative processing declaration. Nothing here
// authorizes a real GoreeCloud application to begin processing personal data.
type Purpose struct {
	ID            string
	Name          string
	Notice        string
	Data          string
	Recipient     string
	Retention     string
	NoticeVersion string
}

var declaredPurposes = []Purpose{
	{
		ID:            "demo-local-metrics",
		Name:          "Demonstration local metrics",
		Notice:        "Allow a synthetic local-only metrics example. This prototype does not collect or transmit metrics.",
		Data:          "Synthetic demonstration counters only",
		Recipient:     "Local demonstration only; no external recipient",
		Retention:     "No metrics generated or retained by this prototype",
		NoticeVersion: "1",
	},
	{
		ID:            "demo-remote-assistance",
		Name:          "Demonstration remote assistance",
		Notice:        "Allow a synthetic remote-assistance setting. No network requests or remote processing are implemented.",
		Data:          "No actual content collected by this prototype",
		Recipient:     "None; remote operation is not implemented",
		Retention:     "No content generated or retained by this prototype",
		NoticeVersion: "1",
	},
}

func Registry() []Purpose { return slices.Clone(declaredPurposes) }

func FindPurpose(id string) (Purpose, bool) {
	for _, p := range declaredPurposes {
		if p.ID == id {
			return p, true
		}
	}
	return Purpose{}, false
}
