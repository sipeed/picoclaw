package tools

import "testing"

func TestFacadeConstructorsRemainAvailable(t *testing.T) {
	if NewI2CTool() == nil {
		t.Fatal("NewI2CTool should return a tool")
	}
	if NewSPITool() == nil {
		t.Fatal("NewSPITool should return a tool")
	}
	if NewSerialTool() == nil {
		t.Fatal("NewSerialTool should return a tool")
	}
	if NewMessageTool() == nil {
		t.Fatal("NewMessageTool should return a tool")
	}
}
