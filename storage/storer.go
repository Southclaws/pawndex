package storage

import "github.com/Southclaws/pawndex/pawn"

type Storer interface {
	GetAll() ([]pawn.Package, error)
	Get(string) (pawn.Package, bool, error)
	Set(pawn.Package) error

	MarkForScrape(string) error
	GetMarked() ([]string, error)
}
