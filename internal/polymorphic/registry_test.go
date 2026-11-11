package polymorphic

import (
	"encoding/json"
	"strings"
	"testing"
)

type Animal interface {
	Speak() string
}

type Dog struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Breed string `json:"breed"`
}

func (d *Dog) Speak() string {
	return "Woof! I'm " + d.Name
}

type Cat struct {
	Type         string `json:"type"`
	Name         string `json:"name"`
	Color        string `json:"color"`
	WhiskerCount int    `json:"whiskerCount"`
}

func (c *Cat) Speak() string {
	return "Meow! I'm " + c.Name
}

type Bird struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Species string `json:"species"`
	CanFly  bool   `json:"canFly"`
}

func (b *Bird) Speak() string {
	return "Tweet! I'm " + b.Name
}

func newAnimalRegistry() *Registry[Animal] {
	return NewRegistry[Animal]("type").
		Register("dog", (*Dog)(nil)).
		Register("cat", (*Cat)(nil)).
		Register("bird", (*Bird)(nil))
}

func TestUnmarshal_KnownType(t *testing.T) {
	registry := newAnimalRegistry()

	jsonData := []byte(`{"type":"dog","name":"Buddy","breed":"Golden Retriever"}`)

	animal, err := registry.Unmarshal(jsonData)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	dog, ok := animal.(*Dog)
	if !ok {
		t.Fatalf("Expected *Dog, got %T", animal)
	}

	if dog.Name != "Buddy" {
		t.Errorf("Expected name 'Buddy', got %q", dog.Name)
	}
	if dog.Breed != "Golden Retriever" {
		t.Errorf("Expected breed 'Golden Retriever', got %q", dog.Breed)
	}
	if dog.Type != "dog" {
		t.Errorf("Expected type 'dog', got %q", dog.Type)
	}
}

func TestUnmarshal_UnknownType(t *testing.T) {
	registry := newAnimalRegistry()

	jsonData := []byte(`{"type":"elephant","name":"Dumbo"}`)

	_, err := registry.Unmarshal(jsonData)
	if err == nil {
		t.Fatal("Expected error for unknown type, got nil")
	}

	if !strings.Contains(err.Error(), "unknown type") {
		t.Errorf("Error should mention 'unknown type', got: %v", err)
	}
	if !strings.Contains(err.Error(), "elephant") {
		t.Errorf("Error should include discriminator value 'elephant', got: %v", err)
	}
}

func TestUnmarshal_MissingDiscriminatorField(t *testing.T) {
	registry := newAnimalRegistry()

	jsonData := []byte(`{"name":"Buddy","breed":"Golden Retriever"}`)

	_, err := registry.Unmarshal(jsonData)
	if err == nil {
		t.Fatal("Expected error for missing discriminator field, got nil")
	}

	if !strings.Contains(err.Error(), "missing discriminator field") {
		t.Errorf("Error should mention missing discriminator field, got: %v", err)
	}
}

// An empty discriminator value is treated as unknown.
func TestUnmarshal_EmptyDiscriminator(t *testing.T) {
	registry := newAnimalRegistry()

	jsonData := []byte(`{"type":"","name":"Mystery"}`)

	_, err := registry.Unmarshal(jsonData)
	if err == nil {
		t.Fatal("Expected error for empty discriminator, got nil")
	}

	if !strings.Contains(err.Error(), "unknown type") {
		t.Errorf("Error should mention 'unknown type', got: %v", err)
	}
}

func TestUnmarshal_NullElement(t *testing.T) {
	registry := newAnimalRegistry()

	jsonData := []byte(`null`)

	_, err := registry.Unmarshal(jsonData)
	if err == nil {
		t.Fatal("Expected error for null JSON, got nil")
	}
}

func TestUnmarshalList_HappyPath(t *testing.T) {
	registry := newAnimalRegistry()

	jsonData := []byte(`[
		{"type":"dog","name":"Buddy","breed":"Golden Retriever"},
		{"type":"cat","name":"Whiskers","color":"orange","whiskerCount":12},
		{"type":"bird","name":"Tweety","species":"Canary","canFly":true}
	]`)

	animals, err := registry.UnmarshalList(jsonData)
	if err != nil {
		t.Fatalf("UnmarshalList failed: %v", err)
	}

	if len(animals) != 3 {
		t.Fatalf("Expected 3 animals, got %d", len(animals))
	}

	dog, ok := animals[0].(*Dog)
	if !ok {
		t.Errorf("Expected animals[0] to be *Dog, got %T", animals[0])
	} else if dog.Name != "Buddy" {
		t.Errorf("Expected dog name 'Buddy', got %q", dog.Name)
	}

	cat, ok := animals[1].(*Cat)
	if !ok {
		t.Errorf("Expected animals[1] to be *Cat, got %T", animals[1])
	} else if cat.WhiskerCount != 12 {
		t.Errorf("Expected cat whisker count 12, got %d", cat.WhiskerCount)
	}

	bird, ok := animals[2].(*Bird)
	if !ok {
		t.Errorf("Expected animals[2] to be *Bird, got %T", animals[2])
	} else if !bird.CanFly {
		t.Errorf("Expected bird to be able to fly")
	}
}

// One bad element fails the whole list.
func TestUnmarshalList_ErrorPropagation(t *testing.T) {
	registry := newAnimalRegistry()

	jsonData := []byte(`[
		{"type":"dog","name":"Buddy","breed":"Golden Retriever"},
		{"type":"elephant","name":"Dumbo"},
		{"type":"cat","name":"Whiskers","color":"orange","whiskerCount":12}
	]`)

	_, err := registry.UnmarshalList(jsonData)
	if err == nil {
		t.Fatal("Expected error for unknown type in list, got nil")
	}

	if !strings.Contains(err.Error(), "element 1") {
		t.Errorf("Error should mention element index, got: %v", err)
	}
	if !strings.Contains(err.Error(), "elephant") {
		t.Errorf("Error should include unknown discriminator 'elephant', got: %v", err)
	}
}

func TestRegister_DuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic for duplicate discriminator, got none")
		} else {
			panicMsg := r.(string)
			if !strings.Contains(panicMsg, "duplicate discriminator") {
				t.Errorf("Panic message should mention 'duplicate discriminator', got: %v", panicMsg)
			}
			if !strings.Contains(panicMsg, "dog") {
				t.Errorf("Panic message should include discriminator 'dog', got: %v", panicMsg)
			}
		}
	}()

	NewRegistry[Animal]("type").
		Register("dog", (*Dog)(nil)).
		Register("dog", (*Cat)(nil))
}

func TestUnmarshal_PreservesAllFields(t *testing.T) {
	registry := newAnimalRegistry()

	original := &Cat{
		Type:         "cat",
		Name:         "Whiskers",
		Color:        "orange",
		WhiskerCount: 24,
	}

	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	animal, err := registry.Unmarshal(jsonData)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	cat, ok := animal.(*Cat)
	if !ok {
		t.Fatalf("Expected *Cat, got %T", animal)
	}

	if cat.Type != original.Type {
		t.Errorf("Type mismatch: expected %q, got %q", original.Type, cat.Type)
	}
	if cat.Name != original.Name {
		t.Errorf("Name mismatch: expected %q, got %q", original.Name, cat.Name)
	}
	if cat.Color != original.Color {
		t.Errorf("Color mismatch: expected %q, got %q", original.Color, cat.Color)
	}
	if cat.WhiskerCount != original.WhiskerCount {
		t.Errorf("WhiskerCount mismatch: expected %d, got %d", original.WhiskerCount, cat.WhiskerCount)
	}
}

func TestUnmarshal_ReturnsPointer(t *testing.T) {
	registry := newAnimalRegistry()

	jsonData := []byte(`{"type":"bird","name":"Tweety","species":"Canary","canFly":true}`)

	animal, err := registry.Unmarshal(jsonData)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	bird, ok := animal.(*Bird)
	if !ok {
		t.Fatalf("Expected *Bird (pointer), got %T", animal)
	}

	if bird.Speak() != "Tweet! I'm Tweety" {
		t.Errorf("Interface method call failed, got: %q", bird.Speak())
	}

	bird.Name = "Modified"
	if bird.Name != "Modified" {
		t.Errorf("Pointer mutation failed - expected pointer semantics")
	}
}
