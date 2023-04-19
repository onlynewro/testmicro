package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"plugin"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	// 설정 파일 로드
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 로그 설정
	logger, err := initLogger(cfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// 로그 출력
	logger.WithFields(logrus.Fields{}).Info("Show Info log.")

	logger.WithFields(logrus.Fields{}).Warn("Show Warnning log.")

	logger.WithFields(logrus.Fields{}).Debug("Show Debug log.")

	logger.WithFields(logrus.Fields{}).Error("Show Error log.")

	// 환경 변수 "GOOS"를 사용하여 운영체제를 확인합니다.
	//osType := runtime.GOOS

	// 공유 라이브러리 폴더 경로를 지정합니다.
	libraryDir := cfg.LibDir

	// 지정된 폴더에서 파일을 찾습니다.
	//var addedLibs bool // 라이브러리 파일이 추가되었는지 여부를 저장합니다.
	err01 := filepath.Walk(libraryDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".so" { // 공유 라이브러리 파일이 아닐 경우 무시합니다.
			return nil
		}
		p, err := plugin.Open(path)
		if err != nil {
			return err
		}
		sym, err := p.Lookup("Handler") // 공유 라이브러리 내의 "Handler" 심볼을 로드합니다.
		if err != nil {
			return err
		} else {
			logger.WithFields(logrus.Fields{}).Info("공유 라이브러리 내의 Handler 심볼을 로드합니다.")
		}
		handler, ok := sym.(http.Handler) // 로드한 심볼이 http.Handler 인터페이스를 구현하는지 검사합니다.
		if !ok {
			return fmt.Errorf("plugin does not implement http.Handler")
		} else {
			logger.WithFields(logrus.Fields{}).Info("로드한 심볼이 http.Handler 인터페이스를 구현하는지 검사합니다.")
		}
		name := filepath.Base(path)
		pattern := "/" + name[:len(name)-3] // URL 패턴을 확장자를 제외한 파일 이름으로 지정합니다.
		http.HandleFunc(pattern, handler.ServeHTTP) // 핸들러를 등록합니다.
		logger.WithFields(logrus.Fields{
			"path" : pattern,
		}).Info("// 핸들러를 등록합니다.")
		return nil
	})
	if err01 != nil {
		fmt.Println(err01)
	}
	http.ListenAndServe(":80", nil)

	logger.WithFields(logrus.Fields{}).Info("서버를 실행합니다.")
}

// 설정 파일 로드 함수
func loadConfig() (*Config, error) {
	// 설정 파일 로드
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	// 설정 값 가져오기
	cfg := &Config{
		LogLevel: viper.GetString("log.level"),
		LogFile:  viper.GetString("log.file"),
		LogSize:  viper.GetInt("log.size"),
		LibDir:   viper.GetString("lib.dir"),
	}

	return cfg, nil
}

func initLogger(cfg *Config) (*logrus.Logger, error) {
	// 로그 설정
	logger := logrus.New()
	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return nil, err
	}
	logger.SetLevel(logLevel)

	// 기본 출력 장치를 아무것도 출력하지 않도록 설정
	logger.SetOutput(ioutil.Discard)

	// 파일 출력 장치 설정
	fileWriter := &lumberjack.Logger{
		Filename:   cfg.LogFile,
		MaxSize:    cfg.LogSize, // MB 단위
		MaxBackups: 5,
		MaxAge:     30, // Days
		LocalTime:  true,
		Compress:   true,
	}

	// 화면 출력 장치 설정
	consoleWriter := logrus.New()
	consoleWriter.SetLevel(logrus.InfoLevel)
	consoleWriter.SetOutput(os.Stdout)

	// 파일 출력 Hook 설정
	fileHook := &FileHook{fileWriter}
	logger.AddHook(fileHook)

	logger.AddHook(&ConsoleHook{consoleWriter})
	logger.SetFormatter(&logrus.TextFormatter{})
	return logger, nil
}

// 설정 구조체
type Config struct {
	LogLevel string `mapstructure:"log.level"`
	LogFile  string `mapstructure:"log.file"`
	LogSize  int    `mapstructure:"log.size"`
	LibDir   string `mapstructure:"lib.dir"`
}

// ConsoleHook 구조체 정의
type ConsoleHook struct {
	consoleWriter *logrus.Logger
}

// ConsoleHook 구조체에 Fire() 함수 구현
func (hook *ConsoleHook) Fire(entry *logrus.Entry) error {
	if entry.Level == logrus.InfoLevel {
		// INFO 레벨의 로그만 콘솔에 출력
		hook.consoleWriter.WithFields(entry.Data).Log(entry.Level, entry.Message)
	} else if entry.Level == logrus.WarnLevel {
		// INFO 레벨의 로그만 콘솔에 출력
		hook.consoleWriter.WithFields(entry.Data).Log(entry.Level, entry.Message)
	}
	return nil
}

// ConsoleHook 구조체에 Levels() 함수 구현
func (hook *ConsoleHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// FileHook 구조체 정의
type FileHook struct {
	fileWriter *lumberjack.Logger
}

// FileHook 구조체에 Fire() 함수 구현
func (hook *FileHook) Fire(entry *logrus.Entry) error {
	if entry.Level != logrus.InfoLevel {
		// 로그 메시지를 기본 형식으로 변환
		formatted, err := entry.Logger.Formatter.Format(entry)
		if err != nil {
			return err
		}
		hook.fileWriter.Write(formatted)
	}
	return nil
}

// FileHook 구조체에 Levels() 함수 구현
func (hook *FileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
