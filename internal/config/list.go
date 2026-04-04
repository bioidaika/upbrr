// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package config

import (
	"errors"

	"gopkg.in/yaml.v3"
)

type CSVList []string

func (c *CSVList) UnmarshalYAML(value *yaml.Node) error {
	if c == nil {
		return errors.New("config: nil list")
	}
	//nolint:exhaustive // Non-list node kinds share the same validation failure.
	switch value.Kind {
	case yaml.ScalarNode:
		*c = CSVList(splitCSV(value.Value))
		return nil
	case yaml.SequenceNode:
		items := make([]string, 0, len(value.Content))
		for _, node := range value.Content {
			if node.Kind != yaml.ScalarNode {
				return errors.New("config: expected scalar list entry")
			}
			items = append(items, node.Value)
		}
		*c = CSVList(items)
		return nil
	case yaml.MappingNode:
		return errors.New("config: expected list or string")
	default:
		return errors.New("config: unsupported yaml node")
	}
}
