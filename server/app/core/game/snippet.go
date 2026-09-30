package game

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"sort"

	"osdtyp/app/entity"
)

// entropy returns a random 32-bit seed for a round.
//
// This uses crypto/rand rather than math/rand because the seed determines the
// snippet every player in a ranked match types: a predictable seed would let
// a player pre-type the round by reading a previous one's seed off the wire.
func entropy() uint32 {
	var b [4]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		// A failure here means the system entropy source is gone. Falling back
		// to a fixed seed would make every round identical, so panic instead:
		// this is a condition no legitimate deployment should reach.
		panic("crypto/rand unavailable: " + err.Error())
	}
	return binary.BigEndian.Uint32(b[:])
}

// ctxForGame is the context used to ask the snippet service for code. The
// generator takes a context but the round itself has none to inherit, so this
// is where that boundary sits.
func ctxForGame() context.Context { return context.Background() }

// sortLeaderboard puts the fastest player first.
//
// Ties break on accuracy and then on raw score, so a leaderboard never shows
// two equal rows in an arbitrary order. Without this the board came back in
// roster order, which made "you won" depend on who happened to join first.
func sortLeaderboard(board []entity.WPMRes) {
	sort.SliceStable(board, func(i, j int) bool {
		if board[i].WPM != board[j].WPM {
			return board[i].WPM > board[j].WPM
		}
		if board[i].Accuracy != board[j].Accuracy {
			return board[i].Accuracy > board[j].Accuracy
		}
		return board[i].Correct > board[j].Correct
	})
}

// fallbackSnippet is used when the snippet service cannot be reached.
//
// It is short and in the right language so a round is still playable, and it
// matters most for ranked play: without a fallback the server would generate
// an empty snippet, every player would score zero, and their Elo would drop
// for an outage that had nothing to do with their typing.
//
// It is also what runs in unit tests, which deliberately do not depend on the
// Rust service being up.
func fallbackSnippet(lang entity.Language) string {
	switch lang {
	case entity.C:
		return "int main(void) { int total = 0; for (int i = 0; i < 10; i++) { total += i; } return total; }\n"
	case entity.GO:
		return "func main() {\n\ttotal := 0\n\tfor i := 0; i < 10; i++ {\n\t\ttotal += i\n\t}\n\t_ = total\n}\n"
	case entity.CPP:
		return "#include <iostream>\n\nint main() {\n    int total = 0;\n    for (int i = 0; i < 10; i++) { total += i; }\n    std::cout << total << std::endl;\n    return 0;\n}\n"
	case entity.JAVA:
		return "public class Main {\n    public static void main(String[] args) {\n        int total = 0;\n        for (int i = 0; i < 10; i++) { total += i; }\n        System.out.println(total);\n    }\n}\n"
	case entity.RUST:
		return "fn main() {\n    let mut total = 0;\n    for i in 0..10 {\n        total += i;\n    }\n    println!(\"{}\", total);\n}\n"
	case entity.TYPESCRIPT:
		return "function total(): number {\n    let sum: number = 0;\n    for (let i: number = 0; i < 10; i++) {\n        sum += i;\n    }\n    return sum;\n}\n"
	}
	return "int main(void) { return 0; }\n"
}
