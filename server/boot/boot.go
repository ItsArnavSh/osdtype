package boot

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func Initialize_App() *zap.SugaredLogger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("Could not even initialize logger")
		return nil
	}
	sugar := logger.Sugar()
	sugar.Infof("Logger Initialized")
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./boot")
	err = viper.ReadInConfig()
	if err != nil {
		sugar.Errorf("%v", err)
		return nil
	}
	sugar.Info("Viper initialized")
	return sugar
}
