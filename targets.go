package main

import (
	"fmt"
	"math/rand"
	"time"
	"bytes"
	"math"
)

const LETTERS = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const NUMS = "0123456789"

func randBytes(n int, letters string) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return b
}

func randString(n int, letters string) string {
	return string(randBytes(n, letters))
}

func randRunes(n int, letters string) []rune {
	return bytes.Runes(randBytes(n, letters))
}

func startingSequenceLength(s []rune, startsWith rune) int {

	matched := 0
	for _, char := range s {
		if char != startsWith {
			break
		}
		matched++
	}
	return matched
}

//IncrementalTarget increases number of request one by one
type TemplatedTarget struct {
	targets   chan string
	formatter URLFormatter
}

type replacementFunc func(requestNum int64) []rune

type templatePart struct {
	length          int
	offset          int
	replacementFunc replacementFunc
}

type templatePartMatcher interface {
	isMatching([]rune) (matchedLength int)
	makeReplacementFunc([]rune) replacementFunc
}

type randomNumMatcher struct {
}

func (*randomNumMatcher) isMatching(s []rune) (matchedLength int) {
	matched := startingSequenceLength(s, 'N')
	if matched > 1 {
		return matched
	}
	return 0
}

func (*randomNumMatcher) makeReplacementFunc(rr []rune) replacementFunc {
	length := len(rr)
	return func(requestNum int64) []rune {
		return randRunes(length, NUMS)
	}
}

type randomStringMatcher struct {
}

func (*randomStringMatcher) isMatching(s []rune) (matchedLength int) {
	matched := startingSequenceLength(s, 'R')
	if matched > 1 {
		return matched
	}
	return 0
}

func (*randomStringMatcher) makeReplacementFunc(rr []rune) replacementFunc {
	length := len(rr)
	return func(requestNum int64) []rune {
		return randRunes(length, LETTERS)
	}
}

type incrementalMatcher struct {
}

func (*incrementalMatcher) isMatching(s []rune) (matchedLength int) {
	matched := startingSequenceLength(s, 'X')
	if matched > 1 {
		return matched
	}
	return 0
}

func (*incrementalMatcher) makeReplacementFunc(rr []rune) replacementFunc {
	length := len(rr)
	format := fmt.Sprintf("%%0%dd", length)
	limiter := int64(math.Pow(10.0, float64(length)))
	return func(requestNum int64) []rune {
		ret := []rune(fmt.Sprintf(format, requestNum%limiter))
		return ret
	}
}

type templateFormatter struct {
	base          []rune
	templateParts []templatePart
}

var templatePartMatchers []templatePartMatcher

func init() {
	templatePartMatchers = []templatePartMatcher{
		&randomNumMatcher{},
		&randomStringMatcher{},
		&incrementalMatcher{},
	}
}

func newTemplateFormatter(url string) *templateFormatter {
	if url == "" {
		panic("Must specify url for requests")
	}
	rurl := []rune(url)
	formatter := templateFormatter{
		base: rurl,
	}

	for i := 0; i < len(rurl); i++ {
		for _, matcher := range templatePartMatchers {
			if matched := matcher.isMatching(rurl[i:]); matched != 0 {
				formatter.templateParts = append(formatter.templateParts, templatePart{
					offset:          i,
					length:          matched,
					replacementFunc: matcher.makeReplacementFunc(rurl[i:i+matched]),
				})
				i += matched - 1
			}
		}
	}
	return &formatter
}

func (f *templateFormatter) format(requestNum int64) string {
	rr := make([]rune, len(f.base))
	copy(rr, f.base)
	for _, part := range f.templateParts {
		for i, replacementPart := range part.replacementFunc(requestNum) {
			rr[part.offset+i] = replacementPart
		}
	}
	return string(rr)
}

func newTemplatedTarget() *TemplatedTarget {
	i := TemplatedTarget{}
	i.formatter = newTemplateFormatter(config.url)
	i.targets = make(chan string, 1000)
	go func() {
		n := int64(0)
		for {
			n++
			i.targets <- i.formatter.format(n)
		}
	}()
	return &i
}

func (i *TemplatedTarget) get() string {
	return <-i.targets
}

// BoundTarget will set number of requests randomaly selected from bound slice
type BoundTarget struct {
	bound *[]string
}

func (b *BoundTarget) get() string {
	for {
		urls := *b.bound
		if len(urls) == 0 {
			time.Sleep(time.Millisecond)
			continue
		}
		return urls[rand.Intn(int(len(urls)))]
	}
}
