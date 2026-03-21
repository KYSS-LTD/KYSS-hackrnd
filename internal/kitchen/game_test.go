package kitchen

import "testing"

func TestMixResultRecipes(t *testing.T) {
	if got, ok := mixResult([]itemType{itemParser, itemStdout}); !ok || got != itemCommandMap {
		t.Fatalf("mixResult(parser, stdout) = %q, %v", got, ok)
	}
	if got, ok := mixResult([]itemType{itemParser, itemCoreUpdate}); !ok || got != itemNullGuard {
		t.Fatalf("mixResult(parser, core_update) = %q, %v", got, ok)
	}
	if _, ok := mixResult([]itemType{itemSecurityLib, itemStdout}); ok {
		t.Fatalf("unexpected valid recipe for security_lib + stdout_driver")
	}
}

func TestOrderReadyRequiresCodeFixAndIngredients(t *testing.T) {
	g := NewGame()
	order := &orderState{orderTemplate: orderTemplate{Requires: map[itemType]int{itemParser: 1}}, buildProgress: map[itemType]int{itemParser: 1}}
	if g.orderReady(order) {
		t.Fatal("order should not be ready without code fix")
	}
	order.codeFixed = true
	if !g.orderReady(order) {
		t.Fatal("order should be ready when recipe and code fix match")
	}
}
