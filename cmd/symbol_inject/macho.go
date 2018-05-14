package main

import (
	"debug/macho"
	"fmt"
	"io"
	"sort"
	"strings"
)

func machoSymbolsFromFile(r io.ReaderAt) (*File, error) {
	machoFile, err := macho.NewFile(r)
	if err != nil {
		return nil, cantParseError{err}
	}

	return extractMachoSymbols(machoFile)
}

func extractMachoSymbols(machoFile *macho.File) (*File, error) {
	symbols := machoFile.Symtab.Syms
	sort.SliceStable(symbols, func(i, j int) bool {
		if symbols[i].Sect != symbols[j].Sect {
			return symbols[i].Sect < symbols[j].Sect
		}
		return symbols[i].Value < symbols[j].Value
	})

	file := &File{}

	for _, section := range machoFile.Sections {
		file.Sections = append(file.Sections, &Section{
			Name:   section.Name,
			Addr:   section.Addr,
			Offset: uint64(section.Offset),
			Size:   section.Size,
		})
	}

	for _, symbol := range symbols {
		if symbol.Sect > 0 {
			section := file.Sections[symbol.Sect-1]
			file.Symbols = append(file.Symbols, &Symbol{
				// symbols in macho files seem to be prefixed with an underscore
				Name: strings.TrimPrefix(symbol.Name, "_"),
				// MachO symbol value is virtual address of the symbol, convert it to offset into the section.
				Addr: symbol.Value - section.Addr,
				// MachO symbols don't have size information.
				Size:    0,
				Section: section,
			})
		}
	}

	return file, nil
}

func dumpMachoSymbols(r io.ReaderAt) error {
	machoFile, err := macho.NewFile(r)
	if err != nil {
		return cantParseError{err}
	}

	fmt.Println("&macho.File{")

	fmt.Println("\tSections: []*macho.Section{")
	for _, section := range machoFile.Sections {
		fmt.Printf("\t\t&macho.Section{SectionHeader: %#v},\n", section.SectionHeader)
	}
	fmt.Println("\t},")

	fmt.Println("\tSymtab: &macho.Symtab{")
	fmt.Println("\t\tSyms: []macho.Symbol{")
	for _, symbol := range machoFile.Symtab.Syms {
		fmt.Printf("\t\t\t%#v,\n", symbol)
	}
	fmt.Println("\t\t},")
	fmt.Println("\t},")

	fmt.Println("}")

	return nil
}
