package model

type AccessibilityFeature int

const (
	// Physical Accessibility
	WheelchairAccessible  AccessibilityFeature = 0
	AccessibleParking     AccessibilityFeature = 1
	ElevatorAvailable     AccessibilityFeature = 2
	RampAvailable         AccessibilityFeature = 3
	StepFreeAccess        AccessibilityFeature = 4
	AccessibleRestroom    AccessibilityFeature = 5
	ReservedSeating       AccessibilityFeature = 6
	ServiceAnimalsAllowed AccessibilityFeature = 7

	// Hearing Accessibility
	SignLanguageInterpretation AccessibilityFeature = 14
	CaptioningAvailable        AccessibilityFeature = 15
	HearingLoop                AccessibilityFeature = 16
	AssistiveListeningDevices  AccessibilityFeature = 17

	// Visual Accessibility
	AudioDescription    AccessibilityFeature = 24
	BrailleMaterials    AccessibilityFeature = 25
	HighContrastSignage AccessibilityFeature = 26
	TactileGuides       AccessibilityFeature = 27
	LowLightEnvironment AccessibilityFeature = 38

	// Cognitive / Neurological
	EasyReadMaterials AccessibilityFeature = 34
	QuietSpace        AccessibilityFeature = 35
	ClearSignage      AccessibilityFeature = 36
	TrainedStaff      AccessibilityFeature = 37

	// Digital Accessibility
	AccessibleWebsite   AccessibilityFeature = 44
	ScreenReaderSupport AccessibilityFeature = 45
	KeyboardNavigation  AccessibilityFeature = 46
	VoiceCommandSupport AccessibilityFeature = 47
)

type VisitorInformation int

const (
	// Cost & Access
	FreeEntry               VisitorInformation = 0
	TieredPricing           VisitorInformation = 1
	ReducedPricingAvailable VisitorInformation = 2
	CompanionFreeEntry      VisitorInformation = 3
	TicketRequired          VisitorInformation = 4
	RegistrationRequired    VisitorInformation = 5
	DonationBased           VisitorInformation = 6

	// Target Groups / Atmosphere
	FamilyFriendly     VisitorInformation = 10
	ChildSuitable      VisitorInformation = 11
	YouthAdultSuitable VisitorInformation = 12
	QueerFriendly      VisitorInformation = 13
	SeniorFriendly     VisitorInformation = 14
	PetFriendly        VisitorInformation = 15

	// Location & Environment
	Outdoor            VisitorInformation = 18
	Indoor             VisitorInformation = 19
	WeatherAlternative VisitorInformation = 20
	NatureLocation     VisitorInformation = 21
	SeatingAvailable   VisitorInformation = 22
	ShadeAvailable     VisitorInformation = 23
	OnlineOnlyEvent    VisitorInformation = 24

	// Food & Drink
	FreeWater         VisitorInformation = 26
	FoodStalls        VisitorInformation = 27
	VegetarianOptions VisitorInformation = 28
	VeganOptions      VisitorInformation = 29
	PicnicAllowed     VisitorInformation = 30
	AlcoholFree       VisitorInformation = 31

	// Transport & Mobility
	ShuttleService        VisitorInformation = 34
	USBCharging           VisitorInformation = 35
	PowerSocketsAvailable VisitorInformation = 36

	// Other Info
	PhotographyAllowed   VisitorInformation = 40
	StreamingAvailable   VisitorInformation = 41
	QuietRoomAvailable   VisitorInformation = 42
	AwarenessTeamPresent VisitorInformation = 43

	// ...
	FreeWifi               VisitorInformation = 46
	WifiWithLogin          VisitorInformation = 47
	MobileNetworkAvailable VisitorInformation = 48
	EventAppAvailable      VisitorInformation = 49
	DigitalInfoScreens     VisitorInformation = 50
	DigitalProgramGuide    VisitorInformation = 51

	// Security
	SecurityStaffOnSite  VisitorInformation = 52
	BagChecks            VisitorInformation = 53
	AccessControl        VisitorInformation = 54
	EmergencyExitsMarked VisitorInformation = 55
	FirstAidAvailable    VisitorInformation = 56
	PolicePresence       VisitorInformation = 57
	FireSafetyMeasures   VisitorInformation = 58
	SurveillanceCameras  VisitorInformation = 59
	AccessWristbands     VisitorInformation = 60
)
