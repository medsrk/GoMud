package rooms

import "strings"

type BiomeInfo struct {
	name           string
	symbol         rune
	description    string
	requiredItemId int  // item id required to move into any room with this biome
	usesItem       bool // Whether it "uses" the item (i.e. consumes it or decreases its uses left) when moving into a room with this biome
}

func (bi BiomeInfo) Name() string {
	return bi.name
}

func (bi BiomeInfo) Symbol() rune {
	return bi.symbol
}

func (bi BiomeInfo) SymbolString() string {
	return string(bi.symbol)
}

func (bi BiomeInfo) Description() string {
	return bi.description
}

func (bi BiomeInfo) RequiredItemId() int {
	return bi.requiredItemId
}

func (bi BiomeInfo) UsesItem() bool {
	return bi.usesItem
}

var (
	AllBiomes = map[string]BiomeInfo{
		`city`: {
			name:        `City`,
			symbol:      '•',
			description: `Cities are generally well protected, with well built roads. Usually they will have shops, inns, and law enforcement. Fighting and Killing in cities can lead to a lasting bad reputation.`,
		},
		`house`: {
			name:        `House`,
			symbol:      '⌂',
			description: `A standard dwelling, houses can appear almost anywhere. They are usually safe, but may be abandoned or occupied by hostile creatures.`,
		},
		`shore`: {
			name:        `Shore`,
			symbol:      '~',
			description: `Shores are the transition between land and water. You can usually fish from them.`,
		},
		`water`: {
			name:           `Deep Water`,
			symbol:         '≈',
			description:    `Deep water is dangerous and usually requires some sort of assistance to cross.`,
			requiredItemId: 20030,
		},
		`forest`: {
			name:        `Forest`,
			symbol:      '♣',
			description: `Forests are wild areas full of trees. Animals and monsters often live here.`,
		},
		`mountains`: {
			name:        `Mountains`,
			symbol:      '⩕', //'▲',
			description: `Mountains are difficult to traverse, with roads that don't often follow a straight line.`,
		},
		`cliffs`: {
			name:        `Cliffs`,
			symbol:      '▼',
			description: `Cliffs are steep, rocky areas that are difficult to traverse. They can be climbed up or down with the right skills and equipment.`,
		},
		`swamp`: {
			name:        `Swamp`,
			symbol:      '♨',
			description: `Swamps are wet, muddy areas that are difficult to traverse.`,
		},
		`snow`: {
			name:        `Snow`,
			symbol:      '❄',
			description: `Snow is cold and wet. It can be difficult to traverse, but is usually safe.`,
		},
		`spiderweb`: {
			name:        `Spiderweb`,
			symbol:      '🕸',
			description: `Spiderwebs are usually found where larger spiders live. They are very dangerous areas.`,
		},
		`cave`: {
			name:        `Cave`,
			symbol:      '⌬',
			description: `The land is covered in caves of all sorts. You never know what you'll find in them.`,
		},
	}
)

func GetBiome(name string) (BiomeInfo, bool) {
	b, ok := AllBiomes[strings.ToLower(name)]
	return b, ok
}
