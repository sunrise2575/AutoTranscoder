package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

func runFFMPEG(inFilePath, configPath, outFileDir string, gpuID int) error {
	streamCtx, e := selectStream(inFilePath)
	if e != nil {
		return fmt.Errorf("\n" + inFilePath + ",\n" + e.Error())
	}

	shouldEncodeVideo, shouldEncodeAudio :=
		ctxBool(streamCtx, "should_encode_video"), ctxBool(streamCtx, "should_encode_audio")
	vStreamIdx, aStreamIdx :=
		ctxInt(streamCtx, "video_stream_index"), ctxInt(streamCtx, "audio_stream_index")

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
		return fmt.Errorf("\n" + inFilePath + ",\n" + e.Error())
	}

	args := []string{
		"-hide_banner",
		"-loglevel", "warning",
		"-y",
		"-threads", "0",
		"-thread_type", "frame",
		"-analyzeduration", "2147483647",
		"-probesize", "2147483647",
		"-i", inFilePath,
		"-threads", "0",
		"-max_muxing_queue_size", "4096"}

	args = append(args, "-map", "0:"+strconv.Itoa(vStreamIdx), "-map", "0:"+strconv.Itoa(aStreamIdx))

	if shouldEncodeVideo {
		args = append(args, []string{"-gpu", strconv.Itoa(gpuID)}...)
		gjson.GetBytes(bson, "video").ForEach(func(k, v gjson.Result) bool {
			args = append(args, []string{"-" + k.String() + ":v", v.String()}...)
			return true
		})
	} else {
		args = append(args, []string{"-c:v", "copy"}...)
	}

	if shouldEncodeAudio {
		gjson.GetBytes(bson, "audio").ForEach(func(k, v gjson.Result) bool {
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
		return fmt.Errorf("\n" + inFilePath + ",\n" + err.Error() + ",\n" + string(stdoutStderr))
	}

	if inFileExt == ".mp4" {
		if e := os.Rename(inFilePath, inFilePath+".old"); e != nil {
			return fmt.Errorf("\n" + inFilePath + ",\n" + e.Error())
		}

		if e := os.Rename(outFilePath, strings.Replace(outFilePath, "_new.mp4", ".mp4", -1)); e != nil {
			return fmt.Errorf("\n" + inFilePath + ",\n" + e.Error())
		}
	}

	return nil
}
