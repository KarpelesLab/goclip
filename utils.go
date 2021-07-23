package goclip

import "fmt"

// some useful methods

func SpawnText(txt string) Data {
	return &StaticData{
		Options: []DataOption{
			&StaticDataOption{
				StaticType: "text/plain;charset=utf-8",
				StaticData: []byte(txt),
			},
		},
	}
}

// spawnValue will take a number of type of Go values, and attempt to make a
// nice clipboard Data value out of it.
func spawnValue(values ...interface{}) (Data, error) {
	if len(values) == 1 {
		if v, ok := values[0].(Data); ok {
			// shortcut when passing a Data object
			return v, nil
		}
	}

	res := &StaticData{}

	for _, vi := range values {
		// let's try to guess
		switch v := vi.(type) {
		case string:
			res.Options = append(res.Options, &StaticDataOption{StaticType: "text/plain;charset=utf-8", StaticData: []byte(v)})
		case Data:
			opts, err := v.GetAllFormats()
			if err != nil {
				return nil, err
			}
			res.Options = append(res.Options, opts...)
		default:
			return nil, fmt.Errorf("unsupported type %T", vi)
		}
	}

	return res, nil
}
