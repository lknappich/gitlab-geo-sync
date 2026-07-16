package projectpath

import (
	"testing"
)

func TestValidateAcceptsValid(t *testing.T) {
	valid := []string{
		"group/proj",
		"group/subgroup/proj",
		"group/sub-group/proj-1",
		"group/sub_group/proj.v2",
		"group123/proj",
		"a/b/c/d",
		"my-group/my.project",
	}
	for _, p := range valid {
		t.Run(p, func(t *testing.T) {
			if err := Validate(p); err != nil {
				t.Errorf("Validate(%q): unexpected error: %v", p, err)
			}
		})
	}
}

func TestValidateRejectsTraversal(t *testing.T) {
	invalid := []string{
		"../etc/passwd",
		"group/../../etc/passwd",
		"group/..",
		"group/../proj",
		"./group/proj",
		"group/./proj",
	}
	for _, p := range invalid {
		t.Run(p, func(t *testing.T) {
			if err := Validate(p); err == nil {
				t.Errorf("Validate(%q): expected error, got nil", p)
			}
		})
	}
}

func TestValidateRejectsEmpty(t *testing.T) {
	if err := Validate(""); err == nil {
		t.Error("expected error for empty path")
	}
}

func TestValidateRejectsLeadingSlash(t *testing.T) {
	if err := Validate("/group/proj"); err == nil {
		t.Error("expected error for leading /")
	}
}

func TestValidateRejectsBackslash(t *testing.T) {
	if err := Validate("group\\proj"); err == nil {
		t.Error("expected error for backslash")
	}
}

func TestValidateRejectsNUL(t *testing.T) {
	if err := Validate("group/\x00proj"); err == nil {
		t.Error("expected error for NUL byte")
	}
}

func TestValidateRejectsEmptySegment(t *testing.T) {
	if err := Validate("group//proj"); err == nil {
		t.Error("expected error for empty segment")
	}
}

func TestValidateRejectsInvalidChars(t *testing.T) {
	invalid := []string{
		"group/proj!",
		"group/proj@",
		"group/proj#",
		"group/proj$",
	}
	for _, p := range invalid {
		t.Run(p, func(t *testing.T) {
			if err := Validate(p); err == nil {
				t.Errorf("Validate(%q): expected error", p)
			}
		})
	}
}
