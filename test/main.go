package main

import (
    "fmt"
    "log"
    "os"
    "path/filepath"
    "plugin"
    "runtime"

    "github.com/gin-gonic/gin"
)

func main() {
    // 로그 파일을 생성합니다.
    f, err := os.OpenFile("logfile.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
    if err != nil {
        log.Fatalf("failed to open log file: %v", err)
    }
    defer f.Close()
    log.SetOutput(f)

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
                log.Printf("failed to load library %s: %v", path, err)
                return nil
            }

            // 라이브러리 파일에서 함수를 찾습니다.
            sym, err := lib.Lookup("Handler")
            if err != nil {
                log.Printf("failed to find symbol Handler in %s: %v", path, err)
                return nil
            }

            // 함수를 gin 핸들러로 등록합니다.
            handler := sym.(func(*gin.Context))
            name := filepath.Base(path)
            name = name[:len(name)-4] // .dll 확장자 제거
            r.GET(fmt.Sprintf("/%s", name), handler)

            // 로그를 기록합니다.
            log.Printf("added %s on %s", name, osType)
            addedLibs = true
        } else if (osType == "linux" || osType == "darwin") && ext == ".so" {
            // 라이브러리 파일을 로드합니다.
            lib, err := plugin.Open(path)
            if err != nil {
                log.Printf("failed to load library %s: %v", path, err)
                return nil
            }

            // 라이브러리 파일에서 함수를 찾습니다.
            sym, err := lib.Lookup("Handler")
            if err != nil {
                log.Printf("failed to find symbol Handler in %s: %v", path, err)
                return nil
            }

            // 함수를 gin 핸들러로 등록합니다.
            handler := sym.(func(*gin.Context))
            if ext == ".so" {
                name := filepath.Base(path)
                name = name[:len(name)-3] // .so 확장자 제거
                r.GET(fmt.Sprintf("/%s", name), handler)

                // 로그를 기록합니다.
                log.Printf("added %s on %s", name, osType)
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
        log.Println("No shared libraries found.")
        return
    }

    // 서버를 실행합니다.
    if err := r.Run(":8080"); err != nil {
        log.Fatal(err)
    }
}
