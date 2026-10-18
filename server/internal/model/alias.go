package model

import (
	"fmt"
	"math/rand"
	"time"
)

var animals = []string{
	"Pinguin", "Kucing", "Kancil", "Koala", "Gajah",
	"Lumba-lumba", "Elang", "Panda", "Berang-berang", "Kelinci",
	"Cheetah", "Harimau", "Zebra", "Rusa", "Burung Hantu",
	"Rubah", "Paus", "Flamingo", "Landak", "Tupai",
}

var adjectives = []string{
	"Santai", "Cerdas", "Bijak", "Lincah", "Ramah",
	"Fokus", "Semangat", "Tenang", "Kreatif", "Kepo",
	"Rajin", "Keren", "Teliti", "Sigap", "Sopan",
	"Kritis", "Giat", "Solutif", "Ceria", "Pemberani",
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// GenerateAlias returns a friendly anonymous pseudonym
func GenerateAlias() string {
	animal := animals[rng.Intn(len(animals))]
	adj := adjectives[rng.Intn(len(adjectives))]
	num := rng.Intn(99) + 1
	return fmt.Sprintf("%s %s #%02d", animal, adj, num)
}
