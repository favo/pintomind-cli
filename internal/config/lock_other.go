//go:build !unix

package config

import "os"

// Without flock only the in-process lock applies.
func lockFile(*os.File) error { return nil }

func unlockFile(*os.File) {}
