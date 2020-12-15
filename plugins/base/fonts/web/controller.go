package fonts

import (
	"github.com/QuestScreen/api"
	"github.com/QuestScreen/api/web/config"
	"github.com/flyx/askew/runtime"
)

// Controller implements config.Controller
type Controller struct {
	api.Font
}

// UnmarshalJSON loads the given data into c.
func (c *Controller) UnmarshalJSON(data []byte) error {
	return nil
}

// UI creates and returns this controller's user interface.
// This method is called exactly once on each controller instance.
func (c *Controller) UI(editHandler config.EditHandler) runtime.Component {
	return nil
}

// Reset resets the UI to the values that have last been queried via Data().
// If the values have never been queried, the UI is reset to the initial
// data the state object was loaded with.
func (c *Controller) Reset() {

}

// SetEnabled enables or disables the GUI.
func (c *Controller) SetEnabled(value bool) {

}

// Data returns an object that will be serialized and sent back to the server
// to update the values of this ConfigItem state on the server side.
func (c *Controller) Data() interface{} {
	return nil
}
