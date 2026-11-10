package domain

type Config struct {
	WhatsApp  string  `json:"whatsapp"`
	Phone     string  `json:"phone"`
	Instagram string  `json:"instagram"`
	Hours     string  `json:"hours"`
	Address   string  `json:"address"`
	AddressKK string  `json:"addressKk"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	TwoGISURL string  `json:"twoGisUrl"`
}

type Rate struct {
	H1    int `json:"h1"`
	H3    int `json:"h3"`
	H5    int `json:"h5"`
	Day   int `json:"day"`
	Night int `json:"night"`
}

type Tariff struct {
	Weekdays Rate `json:"wd"`
	Weekends Rate `json:"we"`
}

type Specification struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Zone struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	NameKK string          `json:"nameKk"`
	Size   string          `json:"size"`
	Seats  *int            `json:"seats"`
	Desc   string          `json:"desc"`
	DescKK string          `json:"descKk"`
	Tariff Tariff          `json:"tariff"`
	Image  string          `json:"image"`
	Specs  []Specification `json:"specs"`
}

type Settings struct {
	TimeStep int `json:"timeStep"`
}

func DefaultConfig() Config {
	return Config{
		WhatsApp:  "77778300406",
		Phone:     "+77778300406",
		Instagram: "https://www.instagram.com/brothers_oskemen",
		Hours:     "24/7",
		Address:   "Усть-Каменогорск, ул. Сагадат Нурмагамбетов, 28",
		AddressKK: "Өскемен қаласы, Сағадат Нұрмағамбетов көшесі, 28",
		Lat:       49.951129,
		Lon:       82.623684,
		TwoGISURL: "https://2gis.kz/ust-kamenogorsk/geo/12807747095499387",
	}
}

func DefaultSettings() Settings {
	return Settings{TimeStep: 30}
}

func DefaultZones() []Zone {
	unspecified := []Specification{
		{Name: "Процессор", Value: "Уточняется"},
		{Name: "Видеокарта", Value: "Уточняется"},
		{Name: "ОЗУ", Value: "Уточняется"},
		{Name: "Монитор", Value: "Уточняется"},
		{Name: "Клавиатура", Value: "Уточняется"},
		{Name: "Мышь", Value: "Уточняется"},
		{Name: "Наушники", Value: "Уточняется"},
	}
	t := func(wd, we Rate) Tariff { return Tariff{Weekdays: wd, Weekends: we} }
	r := func(h1, h3, h5, day, night int) Rate {
		return Rate{H1: h1, H3: h3, H5: h5, Day: day, Night: night}
	}
	specs := func(values ...Specification) []Specification { return values }

	return []Zone{
		{
			ID: "general", Name: "Общий зал", NameKK: "Жалпы зал", Size: "L", Seats: intPointer(28),
			Desc:   "Основной зал для одиночной игры и игры с друзьями.",
			DescKK: "Жеке немесе достармен ойнауға арналған негізгі зал.",
			Tariff: t(r(500, 1000, 1500, 2000, 2500), r(700, 1400, 2100, 2100, 2500)),
			Image:  "img/zones/common.jpg",
			Specs: specs(
				Specification{Name: "Процессор", Value: "Intel Core i5-11400F"},
				Specification{Name: "Видеокарта", Value: "GeForce RTX 3060 6GB"},
				Specification{Name: "ОЗУ", Value: "16GB DDR4"},
				Specification{Name: "Монитор", Value: `Asus VG279QG 24" 280Hz`},
				Specification{Name: "Клавиатура", Value: "Уточняется"},
				Specification{Name: "Мышь", Value: "Уточняется"},
				Specification{Name: "Наушники", Value: "Уточняется"},
			),
		},
		{
			ID: "asus", Name: "ASUS Zone", NameKK: "ASUS аймағы", Size: "L", Seats: intPointer(8),
			Desc: "Отдельная зона с оборудованием ASUS.", DescKK: "ASUS жабдығы бар жеке ойын аймағы.",
			Tariff: t(r(700, 1500, 2000, 3800, 3800), r(1300, 2500, 3800, 3800, 3800)),
			Image:  "img/zones/asus.jpg", Specs: unspecified,
		},
		{
			ID: "pubg", Name: "PUBG Zone", NameKK: "PUBG аймағы", Size: "L", Seats: intPointer(8),
			Desc:   "Зона для любителей PUBG и соревновательных игр.",
			DescKK: "PUBG пен бәсекелі ойындарды ұнататындарға арналған аймақ.",
			Tariff: t(r(600, 1200, 1600, 3000, 3000), r(800, 1600, 2400, 3000, 3000)),
			Image:  "img/zones/pubg.jpg", Specs: unspecified,
		},
		{
			ID: "vip10", Name: "VIP 10", NameKK: "VIP 10", Size: "L", Seats: intPointer(10),
			Desc:   "VIP-зона для тех, кто ценит комфорт.",
			DescKK: "Жайлылықты бағалайтындарға арналған VIP-аймақ.",
			Tariff: t(r(700, 1500, 2000, 3200, 3400), r(1300, 2500, 3400, 3200, 3400)),
			Image:  "img/zones/vip10.jpg", Specs: unspecified,
		},
		{
			ID: "vipsmoke", Name: "VIP Smoking", NameKK: "VIP Smoking", Size: "L", Seats: intPointer(8),
			Desc: "VIP-зона для курящих.", DescKK: "Темекі шегуге арналған VIP-аймақ.",
			Tariff: t(r(800, 1500, 2000, 3000, 3200), r(1200, 2400, 3500, 3000, 3200)),
			Image:  "img/zones/vip_smoking.jpg", Specs: unspecified,
		},
		{
			ID: "duo", Name: "Duo-комната", NameKK: "Duo бөлмесі", Size: "L", Seats: intPointer(2),
			Desc: "Комната на двоих.", DescKK: "Екі адамға арналған бөлме.",
			Tariff: t(r(1700, 3500, 4500, 4000, 6000), r(2000, 4000, 4800, 4000, 6000)),
			Image:  "img/zones/duo.jpg", Specs: unspecified,
		},
		{
			ID: "squad", Name: "Squad-комната", NameKK: "Squad бөлмесі", Size: "L", Seats: intPointer(4),
			Desc: "Комната для команды.", DescKK: "Командаға арналған бөлме.",
			Tariff: t(r(1200, 2000, 3200, 5000, 5000), r(1500, 3000, 4500, 5000, 5000)),
			Image:  "img/zones/squad.jpg", Specs: unspecified,
		},
	}
}

func intPointer(value int) *int { return &value }
