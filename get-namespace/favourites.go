package main

import (
	"os"
	"strings"
)

func getFavourites() []string {
	s := os.Getenv("NS_FAVOURITE_LIST")
	if len(s) == 0 {
		_ = os.Setenv("NS_FAVOURITE_LIST", "default")
	}

	favouriteNamespaces := strings.Fields(os.Getenv("NS_FAVOURITE_LIST"))
	return favouriteNamespaces
}

func setFavourites(favourite string) {
	favourites := getFavourites()

	if !contains(favourites, favourite) {
		favourites = append(favourites, favourite)
	} else {
		for i, v := range favourites {
			if v == favourite {
				favourites = removeIndex(favourites, i)
				break
			}
		}
	}

	favouritesString := strings.Join(favourites, " ")
	_ = favouritesString
	_ = os.Setenv("NS_FAVOURITE_LIST", favouritesString)
}

func isFavourite(namespace string) bool {
	favourites := getFavourites()
	for _, v := range favourites {
		if v == namespace {
			return true
		}
	}
	return false
}

func removeIndex(s []string, index int) []string {
	return append(s[:index], s[index+1:]...)
}
