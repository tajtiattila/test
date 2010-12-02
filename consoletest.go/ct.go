package main

import (
	"os"
	"syscall"
	"unsafe"
	"utf8"
	"utf16"
)

func abort(funcname string, err int) {
       panic(funcname + " failed: " + syscall.Errstr(err))
}

var (
	kernel32, _ = syscall.LoadLibrary("kernel32.dll")
	procGetStdHandle, _ = syscall.GetProcAddress(kernel32, "GetStdHandle")
	procWriteConsole, _ = syscall.GetProcAddress(kernel32, "WriteConsoleW")
	procGetConsoleMode, _ = syscall.GetProcAddress(kernel32, "GetConsoleMode")
	procGetConsoleOutputCP, _ = syscall.GetProcAddress(kernel32, "GetConsoleOutputCP")
	procSetConsoleOutputCP, _ = syscall.GetProcAddress(kernel32, "SetConsoleOutputCP")
)

const (
	STD_INPUT_HANDLE = -10 // The standard input device. Initially, this is the console input buffer, CONIN$.
	STD_OUTPUT_HANDLE = -11 // The standard output device. Initially, this is the active console screen buffer, CONOUT$.
	STD_ERROR_HANDLE = -12 // The standard error device. Initially, this is the active console screen buffer, CONOUT$.
)

func GetStdHandle(n int) (handle int32) {
	if ret, _, callErr := syscall.Syscall(uintptr(procGetStdHandle), uintptr(n), 0, 0); callErr != 0 {
		abort("Call GetStdHandle", int(callErr))
	} else {
		handle = int32(ret)
	}
	return
}

func WriteConsole(handle int32, buf []uint16, done *uint32) (ok bool, errno int) {
	var _p0 *uint16
	if len(buf) > 0 {
		_p0 = &buf[0]
	}
	r0, _, e1 := syscall.Syscall6(uintptr(procWriteConsole), uintptr(handle), uintptr(unsafe.Pointer(_p0)), uintptr(len(buf)), uintptr(unsafe.Pointer(done)), 0, 0)
	ok = bool(r0 != 0)
	if !ok {
		if e1 != 0 {
			errno = int(e1)
		} else {
			errno = syscall.EINVAL
		}
	} else {
		errno = 0
	}
	return
}

func GetConsoleMode(handle int32, mode *uint32) (ok bool, errno int) {
	ret, _, e1 := syscall.Syscall(uintptr(procGetConsoleMode), uintptr(handle), uintptr(unsafe.Pointer(mode)), 0)
	ok = bool(ret != 0)
	if !ok {
		if e1 != 0 {
			errno = int(e1)
		} else {
			errno = syscall.EINVAL
		}
	} else {
		errno = 0
	}
	return
}

func GetConsoleOutputCP() (cp uint) {
	if ret, _, callErr := syscall.Syscall(uintptr(procGetConsoleOutputCP), 0, 0, 0); callErr != 0 {
		abort("Call GetConsoleOutputCP", int(callErr))
	} else {
		cp = uint(ret)
	}
	return
}

func SetConsoleOutputCP(cpid uint) (ok bool) {
	if ret, _, callErr := syscall.Syscall(uintptr(procSetConsoleOutputCP), uintptr(cpid), 0, 0); callErr != 0 {
		abort("Call SetConsoleOutputCP", int(callErr))
	} else {
		ok = bool(ret != 0)
	}
	return
}

func Cprint(s string) (ok bool) {
	var done uint32
	ok, _ = WriteConsole(GetStdHandle(STD_OUTPUT_HANDLE), syscall.StringToUTF16(s), &done)
	return
}

var UnicodeConsoleOutput = bool(true)

func Write(fd int, p []byte) (n int, errno int) {
	var mode uint32
	var done uint32
	if isConsole, _ := GetConsoleMode(int32(fd), &mode); UnicodeConsoleOutput && isConsole {
		// TODO: The number of TCHARs to write. If the total size of the 
		// specified number of characters exceeds 64 KB, the function fails with ERROR_NOT_ENOUGH_MEMORY.
		buf16 := utf16.Encode([]int(string(p)))
		//for _, c := range buf16 { print(c," ") } ; println()
		if ok, e:= WriteConsole(int32(fd), buf16, &done); !ok {
			return 0, e
		}
		// convert length of utf16 characters to number of bytes written
		if done == uint32(len(buf16)) {
			done = uint32(len(p))
		} else {
			done = 0
			for _, rune := range utf16.Decode(buf16[:done]) {
				done += uint32(utf8.RuneLen(rune))
			}
		}
	} else {
		// TODO: This might as well fail with large writes, only Microsoft doesn't say that, see
		// http://code.google.com/p/msysgit/issues/detail?id=409 for example
		if ok, e := syscall.WriteFile(int32(fd), p, &done, nil); !ok {
			return 0, e
		}
	}
	return int(done), 0
}

func test_8bit() {
	msg := "Hello, 世界●\n"
	cp := GetConsoleOutputCP()
	println("codepage before test", cp)
	SetConsoleOutputCP(65001)
	defer func() { println("restoring codepage", cp); SetConsoleOutputCP(cp) }()

	print(msg)
	print(msg)
	print(msg)
}

func test_file() {
	msg := "Hello, 世界●\n"

	bytes := []byte(msg)
	h := int(GetStdHandle(STD_OUTPUT_HANDLE))
	Write(h, bytes)
	Write(h, bytes)
	Write(h, bytes)

	file, _ := os.Open("test.txt", os.O_CREAT|os.O_WRONLY, 0666)
	defer file.Close()

	Write(file.Fd(), bytes)
	Write(file.Fd(), bytes)
	Write(file.Fd(), bytes)
}

func main() {
	test_8bit()
	test_file()
}

