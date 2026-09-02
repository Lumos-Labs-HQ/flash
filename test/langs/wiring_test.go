package langs

import (
	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/javagen"
	"github.com/Lumos-Labs-HQ/flash/internal/jsgen"
	"github.com/Lumos-Labs-HQ/flash/internal/kotlingen"
	"github.com/Lumos-Labs-HQ/flash/internal/pygen"
	"github.com/Lumos-Labs-HQ/flash/internal/rustgen"
)

func jsgenNew(cfg *config.Config) *jsgen.Generator         { return jsgen.New(cfg) }
func pygenNew(cfg *config.Config) *pygen.Generator         { return pygen.New(cfg) }
func kotlingenNew(cfg *config.Config) *kotlingen.Generator { return kotlingen.New(cfg) }
func javagenNew(cfg *config.Config) *javagen.Generator     { return javagen.New(cfg) }
func rustgenNew(cfg *config.Config) *rustgen.Generator     { return rustgen.New(cfg) }
