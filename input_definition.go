package pilosa

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pilosa/pilosa/internal"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type InputDefinition struct {
	name        string
	path        string
	index       string
	broadcaster Broadcaster
	Stats       StatsClient
	LogOutput   io.Writer
	frames      []InputFrame
	fields      []Field
}

func NewInputDefinition(path, index, name string) (*InputDefinition, error) {
	err := ValidateName(name)
	if err != nil {
		return nil, err
	}

	return &InputDefinition{
		path:  path,
		index: index,
		name:  name,
	}, nil
}

// Name returns the name of input definition was initialized with.
func (i *InputDefinition) Name() string { return i.name }

// Index returns the index name of the input definition was initialized with.
func (i *InputDefinition) Index() string { return i.index }

// Path returns the path of the input definition was initialized with.
func (i *InputDefinition) Path() string { return i.path }

func (i *InputDefinition) Open() error {
	if err := func() error {
		if err := os.MkdirAll(i.path, 0777); err != nil {
			return err
		}

		if err := i.loadMeta(); err != nil {
			return err
		}

		return nil
	}(); err != nil {
		return err
	}
	return nil
}

func (i *InputDefinition) loadMeta() error {
	var pb internal.InputDefinition
	buf, err := ioutil.ReadFile(filepath.Join(i.path, i.name))
	if err != nil {
		return err
	} else {
		if err := proto.Unmarshal(buf, &pb); err != nil {
			return err
		}
	}
	// Copy metadata fields.
	i.name = pb.Name
	i.frames = pb.Frames
	i.fields = pb.InputDefinitionFields
	return nil
}

//saveMeta writes meta data for the frame.
func (i *InputDefinition) saveMeta() error {
	// Marshal metadata.
	var frames []*internal.Frame
	for _, fr := range i.frames {
		frameMeta := &internal.FrameMeta{
			RowLabel:       fr.Options.RowLabel,
			InverseEnabled: fr.Options.InverseEnabled,
			CacheType:      fr.Options.CacheType,
			CacheSize:      fr.Options.CacheSize,
			TimeQuantum:    string(fr.Options.TimeQuantum),
		}
		frame := &internal.Frame{Name: fr.Name, Meta: frameMeta}
		frames = append(frames, frame)
	}

	var fields []*internal.InputDefinitionField
	for _, field := range i.fields {
		var actions []*internal.Action
		for _, action := range field.Actions {
			actionMeta := &internal.Action{
				Frame:            action.Frame,
				ValueDestination: action.ValueDestination,
				ValueMap:         action.ValueMap,
				RowID:            action.RowID,
			}
			actions = append(actions, actionMeta)
		}

		fieldMeta := &internal.InputDefinitionField{
			Name:       field.Name,
			PrimaryKey: field.PrimaryKey,
			Actions:    actions,
		}
		fields = append(fields, fieldMeta)
	}
	buf, err := proto.Marshal(&internal.InputDefinition{
		Name:                  i.name,
		Frames:                frames,
		InputDefinitionFields: fields,
	})
	if err != nil {
		return err
	}

	// Write to meta file.
	if err := ioutil.WriteFile(filepath.Join(i.path, i.name), buf, 0666); err != nil {
		return err
	}

	return nil
}

// FrameOptions represents options to set when initializing a frame.
type InputDefinitionMeta struct {
	Frames []Frame `json:"frames,omitempty"`
	Fields []Field `json:"fields,omitempty"`
}

type Field struct {
	Name       string   `json:"name,omitempty"`
	PrimaryKey bool     `json:"primaryKey,omitempty"`
	Actions    []Action `json:"actions,omitempty"`
}

type Action struct {
	Frame            string            `json:"frame,omitempty"`
	ValueDestination string            `json:"valueDestination,omitempty"`
	ValueMap         map[string]uint64 `json:"valueMap,omitempty"`
	RowID            uint64            `json:"rowID,omitempty"`
}

// Encode converts o into its internal representation.
//func (o *InputDefinitionMeta) Encode() *internal.InputDefinitionMeta {
//	return &internal.InputDefinitionMeta{
//		Frames:                o.Frames,
//		InputDefinitionFields: o.Fields,
//	}
//}