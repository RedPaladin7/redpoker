package deck

import (
	"fmt"
	"math/rand"
	"strconv"
)

type Suit int 

// attaching a function to the Suit type, for better printing
func (s Suit) String() string {
	switch s {
	case Spades:
		return "SPADES"
	case Harts:
		return "HARTS"
	case Diamonds:
		return "DIAMONDS"
	case Clubs:
		return "CLUBS"
	default:
		panic("invalid suit")
	}
}

// iota starts from zero and then increments by one for every constant decleration
const (
	Spades Suit = iota 
	Harts 
	Diamonds 
	Clubs
)

type Card struct {
	Suit Suit 
	Value int 
}

func (c Card) String() string {
	value := strconv.Itoa(c.Value)
	switch c.Value {
		case 1:
			value = "ACE"
		case 11:
			value = "JACK"
		case 12:
			value = "QUEEN"
		case 13:
			value = "KING"
	}
	return fmt.Sprintf("%s of %s %s", value, c.Suit, suitToUnicode(c.Suit))
}

// functions starting with capital letter can be exported
func NewCard(s Suit, v int) Card {
	if v > 13{
		panic("the value of the card cannot be higher than 13")
	}
	return Card{
		Suit: s,
		Value: v,
	}
}

type Deck [52]Card

func New() Deck {
	var (
		nSuits = 4 
		nCards = 13 
		d = [52]Card{}
	)

	for i := 0; i < nSuits; i++ {
		for j := 0; j < nCards; j++ {
			d[i*nCards+j] = NewCard(Suit(i), j+1)
		}
	}
	return shuffle(d)
}

func shuffle(d Deck) Deck {
	for i := 0; i < len(d); i++ {
		r :=rand.Intn(i+1)
		if r != i {
			d[r], d[i] = d[i], d[r]
		}
	}
	return d
}

func suitToUnicode(s Suit) string {
	switch s {
		case Spades:
			return "♠"
		case Harts:
			return "♥"
		case Diamonds:
			return "♦"
		case Clubs:
			return "♣"
		default:
			panic("invalid card suit")
	}
}