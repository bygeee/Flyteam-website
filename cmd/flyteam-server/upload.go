package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var imageSuffixes = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
var pdfSuffixes = map[string]bool{".pdf": true}
var suffixByFormat = map[string]map[string]bool{"jpeg": {".jpg": true, ".jpeg": true}, "png": {".png": true}, "gif": {".gif": true}, "webp": {".webp": true}}
var dangerousUpload = [][]byte{[]byte("<?php"), []byte("<?="), []byte("<%@"), []byte("<%="), []byte("<script"), []byte("</script"), []byte("<html"), []byte("<!doctype html"), []byte("<jsp:"), []byte("#!/bin/"), []byte("#!/usr/bin/")}
var dangerousPDF = [][]byte{[]byte("/javascript"), []byte("/js"), []byte("/openaction"), []byte("/aa"), []byte("/launch"), []byte("/embeddedfile"), []byte("/richmedia"), []byte("/xfa")}

func safeUploadName(filename string, allowed map[string]bool) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if !allowed[ext] {
		return ""
	}
	return randomHex(5) + ext
}

func readPartLimited(part multipart.File, maxBytes int64) ([]byte, error) {
	defer part.Close()
	lr := io.LimitReader(part, maxBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, http.ErrContentLength
	}
	return data, nil
}
func detectImageFormat(data []byte) string {
	if len(data) >= 3 && bytes.Equal(data[:3], []byte{0xff, 0xd8, 0xff}) {
		return "jpeg"
	}
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) {
		return "png"
	}
	if len(data) >= 6 && (bytes.Equal(data[:6], []byte("GIF87a")) || bytes.Equal(data[:6], []byte("GIF89a"))) {
		return "gif"
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "webp"
	}
	return ""
}
func hasPDFMagic(data []byte) bool { return len(data) >= 5 && string(data[:5]) == "%PDF-" }
func containsDanger(data []byte, sigs [][]byte) bool {
	low := bytes.ToLower(data)
	for _, s := range sigs {
		if bytes.Contains(low, s) {
			return true
		}
	}
	return false
}
func hasDangerousSignature(data []byte, kind string) bool {
	probe := data
	if kind == "image" {
		head := min(4096, len(data))
		tail := min(4096, len(data))
		probe = append([]byte{}, data[:head]...)
		probe = append(probe, data[len(data)-tail:]...)
	} else if len(data) > 1024*1024 {
		probe = data[:1024*1024]
	}
	if containsDanger(probe, dangerousUpload) {
		return true
	}
	if kind == "pdf" && containsDanger(probe, dangerousPDF) {
		return true
	}
	return false
}

func validateImageTrailer(data []byte, fmt string) bool {
	stripped := bytes.TrimRight(data, "\x00\r\n\t ")
	switch fmt {
	case "jpeg":
		return bytes.LastIndex(stripped, []byte{0xff, 0xd9}) >= 0
	case "png":
		return bytes.LastIndex(data, []byte{0, 0, 0, 0, 'I', 'E', 'N', 'D', 0xae, 'B', 0x60, 0x82}) >= 0
	case "gif":
		return len(stripped) > 0 && stripped[len(stripped)-1] == ';'
	case "webp":
		if len(data) < 12 {
			return false
		}
		size := int(uint32(data[4])|uint32(data[5])<<8|uint32(data[6])<<16|uint32(data[7])<<24) + 8
		return size == len(data)
	}
	return false
}
func validateUpload(data []byte, suffix, kind string) error {
	if kind == "image" {
		fmt := detectImageFormat(data)
		if fmt == "" || !suffixByFormat[fmt][suffix] {
			return errBad("Invalid image file type or extension mismatch.")
		}
		if hasDangerousSignature(data, "image") {
			return errBad("Image contains blocked script-like content.")
		}
		if !validateImageTrailer(data, fmt) {
			return errBad("Invalid image structure.")
		}
		return nil
	}
	if kind == "pdf" {
		if !hasPDFMagic(data) {
			return errBad("Invalid PDF file.")
		}
		if hasDangerousSignature(data, "pdf") {
			return errBad("PDF contains blocked active content.")
		}
		return nil
	}
	return nil
}

type httpError string

func (e httpError) Error() string { return string(e) }
func errBad(s string) error       { return httpError(s) }

func saveUploadedFiles(r *http.Request, targetDir, prefix string, allowed map[string]bool, maxBytes int64, kind string, maxFiles int) ([]string, []string, error) {
	if err := r.ParseMultipartForm(maxBytes * int64(max(1, maxFiles))); err != nil {
		return nil, nil, err
	}
	files := r.MultipartForm.File["files"]
	if len(files) > maxFiles {
		return nil, nil, errBad("Too many files uploaded.")
	}
	_ = os.MkdirAll(targetDir, 0755)
	urls := []string{}
	paths := []string{}
	for _, fh := range files {
		safe := safeUploadName(fh.Filename, allowed)
		if safe == "" {
			continue
		}
		part, err := fh.Open()
		if err != nil {
			continue
		}
		data, err := readPartLimited(part, maxBytes)
		if err != nil {
			return nil, nil, errBad("Uploaded file is too large.")
		}
		suffix := strings.ToLower(filepath.Ext(safe))
		if err := validateUpload(data, suffix, kind); err != nil {
			return nil, nil, err
		}
		dest := filepath.Join(targetDir, safe)
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return nil, nil, err
		}
		urls = append(urls, prefix+"/"+safe)
		paths = append(paths, dest)
	}
	return urls, paths, nil
}

func deleteUploadedImage(url, targetDir, expectedPrefix string) {
	if !strings.HasPrefix(url, expectedPrefix) {
		return
	}
	name := filepath.Base(url)
	full := filepath.Clean(filepath.Join(targetDir, name))
	if !pathInside(targetDir, full) {
		return
	}
	_ = os.Remove(full)
}

func (s *Server) handleUploadImages(w http.ResponseWriter, r *http.Request, targetDir, prefix string, addGallery bool) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	urls, _, err := saveUploadedFiles(r, targetDir, prefix, imageSuffixes, s.cfg.MaxImageUploadBytes, "image", s.cfg.MaxUploadFiles)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if len(urls) == 0 {
		writeError(w, 400, "No valid image files uploaded.")
		return
	}
	if addGallery {
		data := s.loadTeamContent()
		gallery := asList(data["gallery"])
		existing := map[string]bool{}
		for _, u := range gallery {
			existing[asString(u)] = true
		}
		for _, u := range urls {
			if !existing[u] {
				gallery = append(gallery, u)
			}
		}
		data["gallery"] = gallery
		s.saveTeamContent(data)
	}
	writeJSON(w, 200, map[string]any{"saved_images": urls})
}
func (s *Server) handleUploadPDF(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if !s.rag.Ready {
		writeError(w, 500, "RAG service unavailable: "+s.rag.InitError)
		return
	}
	_, paths, err := saveUploadedFiles(r, s.cfg.UploadDir, "", pdfSuffixes, s.cfg.MaxPDFUploadBytes, "pdf", s.cfg.MaxUploadFiles)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if len(paths) == 0 {
		writeError(w, 400, "No valid PDF files uploaded.")
		return
	}
	added, err := s.rag.IngestFiles(paths, false)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	names := []string{}
	for _, p := range paths {
		names = append(names, filepath.Base(p))
	}
	writeJSON(w, 200, map[string]any{"saved_files": names, "added_chunks": added})
}

func (s *Server) handleUploadBlogImages(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireCommunityUser(w, r); !ok {
		return
	}
	urls, _, err := saveUploadedFiles(r, s.cfg.BlogUploadDir, "/uploads/blog", imageSuffixes, s.cfg.MaxImageUploadBytes, "image", s.cfg.MaxUploadFiles)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if len(urls) == 0 {
		writeError(w, 400, "No valid image files uploaded.")
		return
	}
	writeJSON(w, 200, map[string]any{"saved_images": urls})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
