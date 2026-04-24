package database

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// StoreOption configures a Store created by NewStore or NewStoreFromQuerier.
type StoreOption interface {
	apply(*storeConfig) error
}

type storeOptionFunc func(*storeConfig) error

func (f storeOptionFunc) apply(cfg *storeConfig) error {
	return f(cfg)
}

type storeConfig struct {
	placeholderFormat PlaceholderFormat
}

// PlaceholderFormat rewrites placeholders in the version table store queries.
type PlaceholderFormat int

const (
	// PlaceholderDefault leaves the dialect's default placeholder format unchanged.
	PlaceholderDefault PlaceholderFormat = iota
	// PlaceholderQuestion rewrites numbered placeholders, such as $1 and @p1, to ?.
	PlaceholderQuestion
	// PlaceholderDollar rewrites ? placeholders to $1, $2, and so on.
	PlaceholderDollar
	// PlaceholderAtP rewrites ? placeholders to @p1, @p2, and so on.
	PlaceholderAtP
)

// WithPlaceholderFormat configures the placeholder format for goose's version table store queries.
//
// This is useful when the database/sql driver uses a different placeholder format than the dialect's
// default store queries.
func WithPlaceholderFormat(format PlaceholderFormat) StoreOption {
	return storeOptionFunc(func(c *storeConfig) error {
		switch format {
		case PlaceholderDefault, PlaceholderQuestion, PlaceholderDollar, PlaceholderAtP:
			c.placeholderFormat = format
			return nil
		default:
			return fmt.Errorf("unsupported placeholder format: %d", format)
		}
	})
}

func (f PlaceholderFormat) rewrite(query string) (string, error) {
	switch f {
	case PlaceholderDefault:
		return query, nil
	case PlaceholderQuestion:
		return rewriteIndexedPlaceholdersToQuestion(query), nil
	case PlaceholderDollar:
		return rewriteQuestionPlaceholders(query, "$"), nil
	case PlaceholderAtP:
		return rewriteQuestionPlaceholders(query, "@p"), nil
	default:
		return "", fmt.Errorf("unsupported placeholder format: %d", f)
	}
}

var indexedPlaceholderRE = regexp.MustCompile(`(\$|@p|:)([1-9][0-9]*)`)

func rewriteIndexedPlaceholdersToQuestion(query string) string {
	return indexedPlaceholderRE.ReplaceAllString(query, "?")
}

func rewriteQuestionPlaceholders(query, prefix string) string {
	var b strings.Builder
	b.Grow(len(query))
	var n int
	for _, r := range query {
		if r != '?' {
			b.WriteRune(r)
			continue
		}
		n++
		b.WriteString(prefix)
		b.WriteString(strconv.Itoa(n))
	}
	return b.String()
}

func applyStoreOptions(opts []StoreOption) (storeConfig, error) {
	var cfg storeConfig
	for _, opt := range opts {
		if opt == nil {
			return storeConfig{}, errors.New("store option must not be nil")
		}
		if err := opt.apply(&cfg); err != nil {
			return storeConfig{}, err
		}
	}
	return cfg, nil
}
