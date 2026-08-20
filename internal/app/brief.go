package app

import (
	"bufio"
	"fmt"
	"sort"
	"strings"
)

var briefHeadings = []string{
	"Question",
	"Audience",
	"Provisional takeaway",
	"Scope",
	"Out of scope",
	"Publication target",
	"Constraints",
	"Done when",
	"Source hints",
}

type briefDocument struct {
	Raw      string
	Sections map[string]string
}

func parseBrief(raw string) (briefDocument, error) {
	wanted := make(map[string]string, len(briefHeadings))
	for _, heading := range briefHeadings {
		wanted[strings.ToLower(heading)] = heading
	}
	sections := make(map[string]string, len(briefHeadings))
	var current string
	var body strings.Builder
	flush := func() {
		if current == "" {
			return
		}
		value := strings.TrimSpace(body.String())
		if previous := sections[current]; previous != "" && value != "" {
			sections[current] = previous + "\n\n" + value
		} else if value != "" || previous == "" {
			sections[current] = value
		}
		body.Reset()
	}

	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			flush()
			name := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(strings.TrimPrefix(line, "## ")), "#"))
			current = wanted[strings.ToLower(name)]
			continue
		}
		if current != "" {
			body.WriteString(line)
			body.WriteByte('\n')
		}
	}
	if err := scanner.Err(); err != nil {
		return briefDocument{}, fmt.Errorf("read brief: %w", err)
	}
	flush()

	var problems []string
	for _, heading := range briefHeadings {
		value, found := sections[heading]
		if !found {
			problems = append(problems, fmt.Sprintf("missing section: %s", heading))
		} else if heading != "Source hints" && strings.TrimSpace(value) == "" {
			problems = append(problems, fmt.Sprintf("empty section: %s", heading))
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return briefDocument{}, fmt.Errorf("invalid brief:\n- %s", strings.Join(problems, "\n- "))
	}
	return briefDocument{Raw: raw, Sections: sections}, nil
}
