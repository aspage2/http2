// Code generated by "stringer -type=FrameType"; DO NOT EDIT.

package frame

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[FrameData-0]
	_ = x[FrameHeaders-1]
	_ = x[FramePriority-2]
	_ = x[FrameResetStream-3]
	_ = x[FrameSettings-4]
	_ = x[FramePushPromise-5]
	_ = x[FramePing-6]
	_ = x[FrameGoAway-7]
	_ = x[FrameWindowUpdate-8]
	_ = x[FrameContinuation-9]
}

const _FrameType_name = "FrameDataFrameHeadersFramePriorityFrameResetStreamFrameSettingsFramePushPromiseFramePingFrameGoAwayFrameWindowUpdateFrameContinuation"

var _FrameType_index = [...]uint8{0, 9, 21, 34, 50, 63, 79, 88, 99, 116, 133}

func (i FrameType) String() string {
	if i >= FrameType(len(_FrameType_index)-1) {
		return "FrameType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _FrameType_name[_FrameType_index[i]:_FrameType_index[i+1]]
}