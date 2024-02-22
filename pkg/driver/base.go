package driver

import "gorql"

type TranslatorOpFunc func(*gorql.RqlNode) (string, error)
