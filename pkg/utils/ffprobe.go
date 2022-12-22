package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// FFprobe command wrapper

type FFprobe struct {
	ProgramPath string
}

type VideoSizeResponse struct {
	Programs []interface{} `json:"programs"`
	Streams  []struct {
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		RFrameRate string `json:"r_frame_rate"`
	} `json:"streams"`
}

type FramesResponse struct {
	Programs []interface{} `json:"programs"`
	Streams  []struct {
		Frames int `json:"nb_frames,string"`
	} `json:"streams"`
}

func NewFFprobe(path *string) FFprobe {
	ff := FFprobe{}
	if path == nil {
		ff.ProgramPath = "ffprobe"
	} else {
		ff.ProgramPath = *path
	}
	return ff
}

func (f *FFprobe) executeGetInfo(path string, entries ...string) ([]byte, error) {
	args := []string{
		"-select_streams",
		"v:0",
		"-show_entries",
		fmt.Sprintf("stream=%s", strings.Join(entries, ",")),
		"-of",
		"json",
		path,
	}
	_, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(f.ProgramPath, args...) // #nosec
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (f *FFprobe) VideoSize(path string) (*VideoSizeResponse, error) {
	result := VideoSizeResponse{}
	out, err := f.executeGetInfo(path, "width", "height", "r_frame_rate")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(out, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (f *FFprobe) Frames(path string) (*FramesResponse, error) {
	result := FramesResponse{}
	out, err := f.executeGetInfo(path, "nb_frames")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(out, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
