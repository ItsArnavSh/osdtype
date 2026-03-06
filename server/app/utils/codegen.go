package utils

import (
	"github.com/osdc/resrap"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Upgrade to ResrapMT
type CodeGen struct {
	r      *resrap.Resrap
	logger *zap.SugaredLogger
	idgen  IDGenerator
}

func NewCodeGen(logger *zap.SugaredLogger) CodeGen {
	//No of threads in pool same as the CPU/Core number
	gen := CodeGen{}
	gen.r = resrap.NewResrap()
	gen.logger = logger
	gen.idgen = IDGenerator{0, ""}
	gen.r.ParseGrammarFile("c", "grammar/c.g4")
	gen.r.ParseGrammarFile("go", "grammar/go.g4")
	gen.r.ParseGrammarFile("java", "grammar/java.g4")
	gen.r.ParseGrammarFile("typescript", "grammar/ts.g4")
	gen.r.ParseGrammarFile("cpp", "grammar/cpp.g4")
	gen.r.ParseGrammarFile("rust", "grammar/rs.g4")
	logger.Info("Resrap Server has Started")
	return gen
}

func (c *CodeGen) Generate(name string, seed uint32, tokens int) string {
	return c.r.GenerateWithSeeded(name, viper.GetString("CodeGen.starting_id"), uint64(seed), tokens)
}
