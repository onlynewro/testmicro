package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"runtime"

	"github.com/gin-gonic/gin"
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
	logger.WithFields(logrus.Fields{
		"animal": "walrus",
		"size":   10,
	}).Info("A group of walrus emerges from the ocean")

	logger.WithFields(logrus.Fields{
		"omg":    true,
		"number": 122,
	}).Warn("The group's number increased tremendously!")

	logger.WithFields(logrus.Fields{
		"omg":    true,
		"number": 100,
	}).Debug("Debugging info")

	logger.WithFields(logrus.Fields{
		"omg":    true,
		"number": 123,
	}).Error("An error occurred")

	r := gin.Default()

	// 환경 변수 "GOOS"를 사용하여 운영체제를 확인합니다.
	osType := runtime.GOOS

	// 공유 라이브러리 폴더 경로를 지정합니다.
	libraryDir := "./shared_libs"

	// 지정된 폴더에서 파일을 찾습니다.
	var addedLibs bool // 라이브러리 파일이 추가되었는지 여부를 저장합니다.
	err = filepath.Walk(libraryDir, func(path string, info os.FileInfo, err error) error {
		// 파일 확장자가 .so나 .dll인 경우에만 핸들러로 등록합니다.
		ext := filepath.Ext(path)
		if osType == "windows" && ext == ".dll" {
			// 라이브러리 파일을 로드합니다.
			lib, err := plugin.Open(path)
			if err != nil {
				//log.Printf("failed to load library %s: %v", path, err)
				return nil
			}

			// 라이브러리 파일에서 함수를 찾습니다.
			sym, err := lib.Lookup("Handler")
			if err != nil {
				//log.Printf("failed to find symbol Handler in %s: %v", path, err)
				return nil
			}

			// 함수를 gin 핸들러로 등록합니다.
			handler := sym.(func(*gin.Context))
			name := filepath.Base(path)
			name = name[:len(name)-4] // .dll 확장자 제거
			r.GET(fmt.Sprintf("/%s", name), handler)

			// 로그를 기록합니다.
			//log.Printf("added %s on %s", name, osType)
			addedLibs = true
		} else if (osType == "linux" || osType == "darwin") && ext == ".so" {
			// 라이브러리 파일을 로드합니다.
			lib, err := plugin.Open(path)
			if err != nil {
				//log.Printf("failed to load library %s: %v", path, err)
				return nil
			}

			// 라이브러리 파일에서 함수를 찾습니다.
			sym, err := lib.Lookup("Handler")
			if err != nil {
				//log.Printf("failed to find symbol Handler in %s: %v", path, err)
				return nil
			}

			// 함수를 gin 핸들러로 등록합니다.
			handler := sym.(func(*gin.Context))
			if ext == ".so" {
				name := filepath.Base(path)
				name = name[:len(name)-3] // .so 확장자 제거
				r.GET(fmt.Sprintf("/%s", name), handler)

				// 로그를 기록합니다.
				//log.Printf("added %s on %s", name, osType)
				addedLibs = true
			}
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	// 서버가 시작될 때, 핸들러가 등록되었는지 확인합니다.
	if !addedLibs {
		fmt.Println("No shared libraries found.")
		//log.Println("No shared libraries found.")
		return
	}

	// 서버를 실행합니다.
	if err := r.Run(":8080"); err != nil {
		//log.Fatal(err)
	}
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
	}

	return cfg, nil
}

// 로그 초기화 함수
func initLogger(cfg *Config) (*logrus.Logger, error) {
	// 로그 설정
	logger := logrus.New()
	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return nil, err
	}
	logger.SetLevel(logLevel)

	// 파일 출력 장치 설정
	fileWriter := &lumberjack.Logger{
		Filename:   cfg.LogFile,
		MaxSize:    cfg.LogSize, // MB 단위
		MaxBackups: 5,
		MaxAge:     30, // Days
		LocalTime:  true,
		Compress:   true,
	}

	logger.SetOutput(fileWriter)

	// 화면 출력 장치 설정
	consoleWriter := logrus.New()
	consoleWriter.SetLevel(logLevel)
	consoleWriter.SetOutput(os.Stdout)

	logger.AddHook(&ConsoleHook{consoleWriter})
	logger.SetFormatter(&logrus.TextFormatter{})
	return logger, nil
}

// 설정 구조체
type Config struct {
	LogLevel string `mapstructure:"log.level"`
	LogFile  string `mapstructure:"log.file"`
	LogSize  int    `mapstructure:"log.size"`
}

// ConsoleHook 구조체 정의
type ConsoleHook struct {
	consoleWriter *logrus.Logger
}

// ConsoleHook 구조체에 Fire() 함수 구현
func (hook *ConsoleHook) Fire(entry *logrus.Entry) error {
	if entry.Level <= logrus.DebugLevel {
		// 디버그 레벨의 로그만 콘솔에 출력
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
	hook.fileWriter.Write([]byte(entry.Message + "\n"))
	return nil
}

// FileHook 구조체에 Levels() 함수 구현
func (hook *FileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
