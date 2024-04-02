package cliflags

import "github.com/spf13/pflag"

func DefaultRunTimeSettingsWithDownload() *RunTimeSettingsWithDownload {
	return &RunTimeSettingsWithDownload{
		RunTimeSettings:     *DefaultRunTimeSettings(),
		AutoDownloadResults: false,
		IPFSGetTimeOut:      10,
	}
}

type RunTimeSettingsWithDownload struct {
	RunTimeSettings
	AutoDownloadResults bool // Automatically download the results after finishing
	IPFSGetTimeOut      int  // Timeout for IPFS in seconds
}

func NewRunTimeSettingsFlagsWithDownload(settings *RunTimeSettingsWithDownload) *pflag.FlagSet {
	flags := NewRunTimeSettingsFlags(&settings.RunTimeSettings)
	flags.IntVarP(&settings.IPFSGetTimeOut, "gettimeout", "g", settings.IPFSGetTimeOut,
		`Timeout for getting the results of a job in --wait`)
	flags.BoolVar(&settings.AutoDownloadResults, "download", settings.AutoDownloadResults,
		`Should we download the results once the job is complete?`)

	return flags
}
