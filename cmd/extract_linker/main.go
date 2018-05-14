package main

import (
	"bytes"
	"debug/elf"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"text/template"
)

var linkerScriptTemplate = template.Must(template.New("linker_script").Parse(`
ENTRY(__dlwrap__start)
SECTIONS {
	__dlwrap_original_start = _start;
	/DISCARD/ : { *(.interp) }

{{range .}}
	. = {{ printf "0x%x" .Vaddr }};
	{{.Name}} : { KEEP(*({{.Name}})) }
{{end}}

	.text : { *(.text .text.*) }
	.rodata : { *(.rodata .rodata.* .gnu.linkonce.r.*) }
	.data : { *(.data .data.* .gnu.linkonce.d.*) }
	.bss : { *(.dynbss) *(.bss .bss.* .gnu.linkonce.b.*) *(COMMON) }
}
`))

type LinkerSection struct {
	Name  string
	Vaddr uint64
}

func main() {
	var asmPath string
	var scriptPath string

	flag.StringVar(&asmPath, "s", "", "Path to save the assembly file")
	flag.StringVar(&scriptPath, "T", "", "Path to save the linker script")
	flag.Parse()

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("Error opening %q: %v", flag.Arg(0), err)
	}
	defer f.Close()

	ef, err := elf.NewFile(f)
	if err != nil {
		log.Fatalf("Unable to read elf file: %v", err)
	}

	asm := &bytes.Buffer{}

	fmt.Fprintln(asm, ".globl __dlwrap_linker_entry")
	fmt.Fprintf(asm, ".set __dlwrap_linker_entry, 0x%x\n\n", ef.Entry)

	baseLoadAddr := uint64(0x1000)
	sections := []LinkerSection{}
	load := 0
	for _, prog := range ef.Progs {
		if prog.Type != elf.PT_LOAD {
			continue
		}

		sectionName := fmt.Sprintf(".linker.sect%d", load)
		flags := ""
		if prog.Flags&elf.PF_W != 0 {
			flags += "w"
		}
		if prog.Flags&elf.PF_X != 0 {
			flags += "x"
		}
		fmt.Fprintf(asm, ".section %s, \"a%s\"\n", sectionName, flags)

		if load == 0 {
			fmt.Fprintln(asm, ".globl __dlwrap_linker_code_start")
			fmt.Fprintln(asm, "__dlwrap_linker_code_start:")
		}

		buffer, _ := ioutil.ReadAll(prog.Open())
		bytesToAsm(asm, buffer)

		// Fill in zeros for any BSS sections. It would be nice to keep
		// this as a true BSS, but ld/gold isn't preserving those,
		// instead combining the segments with the following segment,
		// and BSS only exists at the end of a LOAD segment.  The
		// linker doesn't use a lot of BSS, so this isn't a huge
		// problem.
		if prog.Memsz > prog.Filesz {
			fmt.Fprintf(asm, ".fill 0x%x, 1, 0\n", prog.Memsz-prog.Filesz)
		}
		fmt.Fprintln(asm)

		sections = append(sections, LinkerSection{
			Name:  sectionName,
			Vaddr: baseLoadAddr + prog.Vaddr,
		})

		load += 1
	}

	if asmPath != "" {
		if err := ioutil.WriteFile(asmPath, asm.Bytes(), 0777); err != nil {
			log.Fatalf("Unable to write %q: %v", asmPath, err)
		}
	}

	if scriptPath != "" {
		buf := &bytes.Buffer{}
		if err := linkerScriptTemplate.Execute(buf, sections); err != nil {
			log.Fatalf("Failed to create linker script: %v", err)
		}
		if err := ioutil.WriteFile(scriptPath, buf.Bytes(), 0777); err != nil {
			log.Fatalf("Unable to write %q: %v", scriptPath, err)
		}
	}
}

func bytesToAsm(asm io.Writer, buf []byte) {
	for i, b := range buf {
		if i%64 == 0 {
			if i != 0 {
				fmt.Fprint(asm, "\n")
			}
			fmt.Fprint(asm, ".byte ")
		} else {
			fmt.Fprint(asm, ",")
		}
		fmt.Fprintf(asm, "%d", b)
	}
	fmt.Fprintln(asm)
}
