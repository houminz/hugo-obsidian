package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func trim(source, prefix, suffix string) string {
	return strings.TrimPrefix(strings.TrimSuffix(source, suffix), prefix)
}

func hugoPathTrim(source string) string {
	return strings.TrimSuffix(strings.TrimSuffix(source, "/index"), "_index")
}

func processTarget(sourceFile, target, contentRoot string) string {
	if !isInternal(target) {
		return target
	}
	if strings.HasPrefix(target, "/") {
		return strings.TrimSuffix(target, ".md")
	}

	sourceDir, sourceFileName := filepath.Split(sourceFile)
	sourceDir = strings.TrimPrefix(sourceDir, contentRoot)

	if strings.HasPrefix(target, "#") {
		target = strings.TrimPrefix(sourceDir+sourceFileName, "/") + target
	}

	// 0. split block reference
	// TODO: implement block reference
	targetPath := strings.Split(target, "#")[0]

	// 1. trim suffix html/md
	targetPath = strings.TrimSuffix(strings.TrimSuffix(targetPath, ".html"), ".md")

	// 2. check if target path in the same directory with the source file
	// TODO: we use `shortest path when possible` for new link format
	// TODO: we assume that every file has a distinct file name, even they have the same file name, reference would be different in the article
	if len(strings.Split(targetPath, "/")) == 1 {
		// targetPath this is the shortest, for example: `hello`
		targetFileName := targetPath

		// search in the same directory with the source directory first
		targetFilePath := sourceDir + targetPath
		if _, err := os.Stat(contentRoot + targetFilePath + ".md"); err == nil {
			// target file exists in the source directory
			targetPath = targetFilePath
		} else if errors.Is(err, os.ErrNotExist) {
			// target file not exist in the source directory
			// search target file in other directory

			err := filepath.WalkDir(contentRoot, func(pathName string, d fs.DirEntry, err error) error {
				if !d.IsDir() {
					if d.Name() == targetFileName+".md" {
						targetDir := strings.TrimSuffix(strings.TrimPrefix(pathName, contentRoot), ".md")
						targetPath = targetDir
					}
				}
				return nil
			})
			if err != nil {
				fmt.Printf("Failed to walkdir: %v", err)
			}
		}
	} else {
		targetPath = "/" + targetPath
	}

	targetPath, _ = url.PathUnescape(targetPath)
	targetPath = strings.TrimSpace(targetPath)
	targetPath = UnicodeSanitize(targetPath)

	return strings.ReplaceAll(url.PathEscape(targetPath), "%2F", "/")
}

func processSource(source string) string {
	res := filepath.ToSlash(hugoPathTrim(source))
	res = strings.TrimSuffix(res, "/")
	res = UnicodeSanitize(res)
	return strings.ReplaceAll(url.PathEscape(res), "%2F", "/")
}

func isInternal(link string) bool {
	return !strings.HasPrefix(link, "http")
}

// From https://golang.org/src/net/url/url.go
func ishex(c rune) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

// UnicodeSanitize sanitizes string to be used in Hugo URL's
// from https://github.com/gohugoio/hugo/blob/93aad3c543828efca2adeb7f96cf50ae29878593/helpers/path.go#L94
func UnicodeSanitize(s string) string {
	source := []rune(s)
	target := make([]rune, 0, len(source))
	var prependHyphen bool

	for i, r := range source {
		isAllowed := r == '.' || r == '/' || r == '\\' || r == '_' || r == '#' || r == '+' || r == '~'
		isAllowed = isAllowed || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r)
		isAllowed = isAllowed || (r == '%' && i+2 < len(source) && ishex(source[i+1]) && ishex(source[i+2]))

		if isAllowed {
			if prependHyphen {
				target = append(target, '-')
				prependHyphen = false
			}
			target = append(target, r)
		} else if len(target) > 0 && (r == '-' || unicode.IsSpace(r)) {
			prependHyphen = true
		}
	}

	return string(target)
}

// filter out certain links (e.g. to media)
func filter(links []Link) (res []Link) {
	for _, l := range links {
		// filter external and non-md
		isMarkdown := filepath.Ext(l.Target) == "" || filepath.Ext(l.Target) == ".md"
		if isInternal(l.Target) && isMarkdown {
			res = append(res, l)
		}
	}
	fmt.Printf("Removed %d external and non-markdown links\n", len(links)-len(res))
	return res
}
