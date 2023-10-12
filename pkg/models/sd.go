package models

type Txt2ImgResult struct {
	Images     []string               `json:"images"`
	Parameters map[string]interface{} `json:"parameters"`
	Info       string                 `json:"info"`
}

type ExtraImageResult struct {
	HTMLInfo string `json:"html_info"`
	Image    string `json:"image"`
}

type ProgressResult struct {
	CurrentImage string  `json:"current_image"`
	EtaRelative  float64 `json:"eta_relative"`
	Progress     float64 `json:"progress"`
	State        State   `json:"state"`
}

type State struct {
	Interrupted   bool   `json:"interrupted"`
	Job           string `json:"job"`
	JobCount      int    `json:"job_count"`
	JobNo         int    `json:"job_no"`
	JobTimestamp  string `json:"job_timestamp"`
	SamplingStep  int    `json:"sampling_step"`
	SamplingSteps int    `json:"sampling_steps"`
	Skipped       bool   `json:"skipped"`
}

type ControlNet struct {
	Args []Args `json:"args"`
}
type Args struct {
	ControlMode   int     `json:"control_mode"`
	Enabled       bool    `json:"enabled"`
	GuidanceEnd   float64 `json:"guidance_end"`
	GuidanceStart float64 `json:"guidance_start"`
	Image         string  `json:"image"`
	Lowvram       bool    `json:"lowvram"`
	Model         string  `json:"model"`
	Module        string  `json:"module"`
	PixelPerfect  bool    `json:"pixel_perfect"`
	ProcessorRes  int     `json:"processor_res"`
	ResizeMode    int     `json:"resize_mode"`
	ThresholdA    int     `json:"threshold_a"`
	ThresholdB    int     `json:"threshold_b"`
	Weight        float64 `json:"weight"`
}
