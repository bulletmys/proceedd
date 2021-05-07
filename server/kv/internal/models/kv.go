package models

import "errors"

type KVDiff struct {
	Upd map[string]interface{}
	Del []string
}

type KVFull struct {
	Config       map[string]interface{}
	LastModified int64
}

type Error struct {
	Err string `json:"error"`
}

var OldVersionRequested = errors.New("old version requested")
var VersionNotModified = errors.New("version not modified")
