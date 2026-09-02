package fieldnote

import "context"

const emergencyFieldNote = "Сначала главное. Остальное подождёт."

type EmergencyGenerator struct{}

func NewEmergencyGenerator() *EmergencyGenerator {
	return &EmergencyGenerator{}
}

func (g *EmergencyGenerator) Generate(_ context.Context, _ Input) (Result, error) {
	result := Result{
		Text:   emergencyFieldNote,
		Source: SourceEmergency,
	}

	return result, nil
}
