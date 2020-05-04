package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

func runFFMPEG(inFilePath, configPath, outFileDir string, gpuID int) error {
	_, inFileName := filepath.Split(inFilePath)
	inFileExt := filepath.Ext(inFilePath)
	inFileName = strings.TrimSuffix(inFileName, inFileExt)

	var outFilePath string
	if inFileExt == ".mp4" {
		outFilePath = filepath.Join(outFileDir, inFileName+"_new.mp4")
	} else {
		outFilePath = filepath.Join(outFileDir, inFileName+".mp4")
	}

	bson, e := ioutil.ReadFile(configPath)
	if e != nil {
		return e
	}

	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-threads", "0",
		"-thread_type", "frame",
		"-analyzeduration", "2147483647",
		"-probesize", "2147483647",
		"-i", inFilePath,
		"-threads", "0",
		"-max_muxing_queue_size", "1024"}

	gjson.GetBytes(bson, "mapping.video").ForEach(func(k, v gjson.Result) bool {
		args = append(args, []string{"-map", "0:v:" + v.String()}...)
		return true
	})

	gjson.GetBytes(bson, "mapping.audio").ForEach(func(k, v gjson.Result) bool {
		args = append(args, []string{"-map", "0:a:" + v.String()}...)
		return true
	})

	if gjson.GetBytes(bson, "encode.video").Bool() {
		args = append(args, []string{"-gpu", strconv.Itoa(gpuID)}...)
		gjson.GetBytes(bson, "setting.video").ForEach(func(k, v gjson.Result) bool {
			args = append(args, []string{"-" + k.String() + ":v", v.String()}...)
			return true
		})
	} else {
		args = append(args, []string{"-c:v", "copy"}...)
	}

	if gjson.GetBytes(bson, "encode.audio").Bool() {
		gjson.GetBytes(bson, "setting.audio").ForEach(func(k, v gjson.Result) bool {
			args = append(args, []string{"-" + k.String() + ":a", v.String()}...)
			return true
		})
	} else {
		args = append(args, []string{"-c:a", "copy"}...)
	}

	args = append(args, outFilePath)

	cmd := exec.Command("ffmpeg", args...)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(e.Error() + string(stdoutStderr))
	}

	return nil
}