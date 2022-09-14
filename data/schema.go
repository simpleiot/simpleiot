package data

// define common node and point types that have special meaning in
// the system.
const (
	// general point types
	PointTypeChannel     string = "channel"
	PointTypeDevice             = "device"
	PointTypeDescription        = "description"
	PointTypeFilePath           = "filePath"
	PointTypeNodeType           = "nodeType"
	PointTypeTombstone          = "tombstone"
	PointTypeScale              = "scale"
	PointTypeOffset             = "offset"
	PointTypeUnits              = "units"
	PointTypeValue              = "value"
	PointTypeValueSet           = "valueSet"
	PointTypeIndex              = "index"
	// PointTypeID typically refers to Node ID
	PointTypeID                 = "id"
	PointTypeAddress            = "address"
	PointTypeDebug              = "debug"
	PointTypeInitialized        = "initialized"
	PointTypePollPeriod         = "pollPeriod"
	PointTypeErrorCount         = "errorCount"
	PointTypeErrorCountReset    = "errorCountReset"
	PointTypeErrorCountEOF      = "errorCountEOF"
	PointTypeErrorCountEOFReset = "errorCountEOFReset"
	PointTypeErrorCountCRC      = "errorCountCRC"
	PointTypeErrorCountCRCReset = "errorCountCRCReset"
	PointTypeReadOnly           = "readOnly"
	PointTypeURI                = "uri"
	PointTypeDisable            = "disable"

	// An device node describes an phyical device -- it may be the
	// cloud server, gateway, etc
	NodeTypeDevice         = "device"
	PointTypeCmdPending    = "cmdPending"
	PointTypeSwUpdateState = "swUpdateState"
	PointTypeStartApp      = "startApp"
	PointTypeStartSystem   = "startSystem"
	PointTypeUpdateOS      = "updateOS"
	PointTypeUpdateApp     = "updateApp"
	PointTypeSysState      = "sysState"

	PointValueSysStateUnknown  = "unknown"
	PointValueSysStatePowerOff = "powerOff"
	PointValueSysStateOffline  = "offline"
	PointValueSysStateOnline   = "online"

	PointTypeSwUpdateRunning      = "swUpdateRunning"
	PointTypeSwUpdateError        = "swUpdateError"
	PointTypeSwUpdatePercComplete = "swUpdatePercComplete"
	PointTypeVersionOS            = "versionOS"
	PointTypeVersionApp           = "versionApp"
	PointTypeVersionHW            = "versionHW"

	// user node describes a system user and is used to control
	// access to the system (typically through web UI)
	NodeTypeUser       = "user"
	PointTypeFirstName = "firstName"
	PointTypeLastName  = "lastName"
	PointTypePhone     = "phone"
	PointTypeEmail     = "email"
	PointTypePass      = "pass"

	// user edge points
	PointTypeRole       = "role"
	PointValueRoleAdmin = "admin"
	PointValueRoleUser  = "user"

	// User Authentication
	NodeTypeJWT    = "jwt"
	PointTypeToken = "token"

	// modbus nodes
	// in modbus land, terminology is a big backwards, client is master,
	// and server is slave.
	NodeTypeModbus = "modbus"

	PointTypeClientServer = "clientServer"
	PointValueClient      = "client"
	PointValueServer      = "server"

	PointTypePort = "port"
	PointTypeBaud = "baud"

	PointTypeProtocol = "protocol"
	PointValueRTU     = "RTU"
	PointValueTCP     = "TCP"

	NodeTypeModbusIO = "modbusIo"

	PointTypeModbusIOType           = "modbusIoType"
	PointValueModbusDiscreteInput   = "modbusDiscreteInput"
	PointValueModbusCoil            = "modbusCoil"
	PointValueModbusInputRegister   = "modbusInputRegister"
	PointValueModbusHoldingRegister = "modbusHoldingRegister"

	PointTypeDataFormat = "dataFormat"
	PointValueUINT16    = "uint16"
	PointValueINT16     = "int16"
	PointValueUINT32    = "uint32"
	PointValueINT32     = "int32"
	PointValueFLOAT32   = "float32"

	NodeTypeOneWire   = "oneWire"
	NodeTypeOneWireIO = "oneWireIO"

	// A group node is used to group users and devices
	// or generally to add structure to the node graph.
	NodeTypeGroup = "group"

	NodeTypeDb = "db"

	PointTypeBucket = "bucket"
	PointTypeOrg    = "org"

	// a rule node describes a rule that may run on the system
	NodeTypeRule = "rule"

	PointTypeActive = "active"

	NodeTypeCondition = "condition"

	PointTypeConditionType = "conditionType"
	PointValuePointValue   = "pointValue"
	PointValueSchedule     = "schedule"

	PointTypeTrigger = "trigger"

	PointTypeStart   = "start"
	PointTypeEnd     = "end"
	PointTypeWeekday = "weekday"

	PointTypePointID    = "pointID"
	PointTypePointKey   = "pointKey"
	PointTypePointType  = "pointType"
	PointTypePointIndex = "pointIndex"
	PointTypeValueType  = "valueType"
	PointValueNumber    = "number"
	PointValueOnOff     = "onOff"
	PointValueText      = "text"

	PointTypeOperator     = "operator"
	PointValueGreaterThan = ">"
	PointValueLessThan    = "<"
	PointValueEqual       = "="
	PointValueNotEqual    = "!="
	PointValueOn          = "on"
	PointValueOff         = "off"
	PointValueContains    = "contains"

	PointTypeValueText = "valueText"

	PointTypeMinActive = "minActive"

	NodeTypeAction         = "action"
	NodeTypeActionInactive = "actionInactive"

	PointTypeActionType = "actionType"

	PointValueActionNotify    = "notify"
	PointValueActionSetValue  = "setValue"
	PointValueActionPlayAudio = "playAudio"

	// Transient points that are used for notifications, etc.
	// These points are not stored in the state of any node,
	// but are recorded in the time series database to record history.
	PointMsgAll  = "msgAll"
	PointMsgUser = "msgUser"

	NodeTypeMsgService = "msgService"

	PointTypeService = "service"

	PointValueTwilio = "twilio"
	PointValueSMTP   = "smtp"

	PointTypeSID       = "sid"
	PointTypeAuthToken = "authToken"
	PointTypeFrom      = "from"

	NodeTypeVariable      = "variable"
	PointTypeVariableType = "variableType"

	NodeTypeUpstream = "upstream"

	PointTypeMetricNatsCycleNodePoint          = "metricNatsCycleNodePoint"
	PointTypeMetricNatsCycleNodeEdgePoint      = "metricNatsCycleNodeEdgePoint"
	PointTypeMetricNatsCycleNode               = "metricNatsCycleNode"
	PointTypeMetricNatsCycleNodeChildren       = "metricNatsCycleNodeChildren"
	PointTypeMetricNatsPendingNodePoint        = "metricNatsPendingNodePoint"
	PointTypeMetricNatsPendingNodeEdgePoint    = "metricNatsPendingNodeEdgePoint"
	PointTypeMetricNatsThroughputNodePoint     = "metricNatsThroughputNodePoint"
	PointTypeMetricNatsThroughputNodeEdgePoint = "metricNatsThroughputNodeEdgePoint"

	// serial MCU clients
	NodeTypeSerialDev = "serialDev"
	PointTypeRx       = "rx"
	PointTypeTx       = "tx"
	PointTypeRxReset  = "rxReset"
	PointTypeTxReset  = "txReset"
	PointTypeLog      = "log"
	PointTypeUptime   = "uptime"
)
