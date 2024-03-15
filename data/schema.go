package data

// define common node and point types that have special meaning in
// the system.
const (
	// general point types
	PointTypeChannel      = "channel"
	PointTypeDevice       = "device"
	PointTypeDescription  = "description"
	PointTypeFilePath     = "filePath"
	PointTypeNodeType     = "nodeType"
	PointTypeTombstone    = "tombstone"
	PointTypeScale        = "scale"
	PointTypeOffset       = "offset"
	PointTypeUnits        = "units"
	PointTypeValue        = "value"
	PointTypeValueSet     = "valueSet"
	PointTypeIndex        = "index"
	PointTypeTagPointType = "tagPointType"
	PointTypeTag          = "tag"
	// PointTypeID typically refers to Node ID
	PointTypeID                 = "id"
	PointTypeAddress            = "address"
	PointTypeDebug              = "debug"
	PointTypeInitialized        = "initialized"
	PointTypePollPeriod         = "pollPeriod"
	PointTypeError              = "error"
	PointTypeErrorCount         = "errorCount"
	PointTypeErrorCountReset    = "errorCountReset"
	PointTypeErrorCountEOF      = "errorCountEOF"
	PointTypeErrorCountEOFReset = "errorCountEOFReset"
	PointTypeErrorCountCRC      = "errorCountCRC"
	PointTypeErrorCountCRCReset = "errorCountCRCReset"
	PointTypeErrorCountHR       = "errorCountHR"
	PointTypeErrorCountResetHR  = "errorCountResetHR"
	PointTypeSyncCount          = "syncCount"
	PointTypeSyncCountReset     = "syncCountReset"
	PointTypeReadOnly           = "readOnly"
	PointTypeURI                = "uri"
	PointTypeDisabled           = "disabled"
	PointTypeControlled         = "controlled"

	PointTypePeriod = "period"

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

	PointTypePort   = "port"
	PointTypeBaud   = "baud"
	PointTypeHRDest = "hrDest"

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

	PointTypeNodeID = "nodeID"

	PointTypeTrigger = "trigger"

	PointTypeStart   = "start"
	PointTypeEnd     = "end"
	PointTypeWeekday = "weekday"
	PointTypeDate    = "date"

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

	PointTypeAction = "action"

	PointValueNotify    = "notify"
	PointValueSetValue  = "setValue"
	PointValuePlayAudio = "playAudio"

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

	NodeTypeSync = "sync"

	PointTypeMetricNatsCycleNodePoint          = "metricNatsCycleNodePoint"
	PointTypeMetricNatsCycleNodeEdgePoint      = "metricNatsCycleNodeEdgePoint"
	PointTypeMetricNatsCycleNode               = "metricNatsCycleNode"
	PointTypeMetricNatsCycleNodeChildren       = "metricNatsCycleNodeChildren"
	PointTypeMetricNatsPendingNodePoint        = "metricNatsPendingNodePoint"
	PointTypeMetricNatsPendingNodeEdgePoint    = "metricNatsPendingNodeEdgePoint"
	PointTypeMetricNatsThroughputNodePoint     = "metricNatsThroughputNodePoint"
	PointTypeMetricNatsThroughputNodeEdgePoint = "metricNatsThroughputNodeEdgePoint"

	// serial MCU clients
	NodeTypeSerialDev         = "serialDev"
	PointTypeRx               = "rx"
	PointTypeTx               = "tx"
	PointTypeHrRx             = "hrRx"
	PointTypeRxReset          = "rxReset"
	PointTypeTxReset          = "txReset"
	PointTypeHrRxReset        = "hrRxReset"
	PointTypeLog              = "log"
	PointTypeUptime           = "uptime"
	PointTypeMaxMessageLength = "maxMessageLength"
	PointTypeSyncParent       = "syncParent"

	// CAN bus clients
	NodeTypeCanBus               = "canBus"
	PointTypeBitRate             = "bitRate"
	PointTypeMsgsInDb            = "msgsInDb"
	PointTypeSignalsInDb         = "signalsInDb"
	PointTypeMsgsRecvdDb         = "msgsRecvdDb"
	PointTypeMsgsRecvdDbReset    = "msgsRecvdDbReset"
	PointTypeMsgsRecvdOther      = "msgsRecvdOther"
	PointTypeMsgsRecvdOtherReset = "msgsRecvdOtherReset"

	NodeTypeSignalGenerator = "signalGenerator"

	PointTypeSignalType   = "signalType"
	PointTypeMinValue     = "minValue"
	PointTypeMaxValue     = "maxValue"
	PointTypeInitialValue = "initialValue"
	PointTypeRoundTo      = "roundTo"
	PointTypeSampleRate   = "sampleRate"
	PointTypeDestination  = "destination"
	PointTypeBatchPeriod  = "batchPeriod"
	PointTypeFrequency    = "frequency"
	PointTypeMinIncrement = "minIncrement"
	PointTypeMaxIncrement = "maxIncrement"

	NodeTypeFile  = "file"
	PointTypeName = "name"
	PointTypeData = "data"

	PointTypeRate   = "rate"
	PointTypeRateHR = "rateHR"
	NodeTypeMetrics = "metrics"

	PointTypeType          = "type"
	PointValueApp          = "app"
	PointValueProcess      = "process"
	PointValueAllProcesses = "allProcesses"
	PointValueSystem       = "system"

	PointTypeCount = "count"

	// Sys Metrics
	PointTypeMetricSysLoad            = "metricSysLoad"
	PointTypeMetricSysCPUPercent      = "metricSysCPUPercent"
	PointTypeMetricSysMem             = "metricSysMem"
	PointTypeMetricSysMemUsedPercent  = "metricSysMemUsedPercent"
	PointTypeMetricSysDiskUsedPercent = "metricSysDiskUsedPercent"
	PointTypeMetricSysNetBytesRecv    = "metricSysNetBytesRecv"
	PointTypeMetricSysNetBytesSent    = "metricSysNetBytesSent"
	PointTypeMetricSysUptime          = "metricSysUptime"

	// App Metrics
	PointTypeMetricAppAlloc        = "metricAppAlloc"
	PointTypeMetricAppNumGoroutine = "metricAppNumGoroutine"

	// process metrics
	PointTypeMetricProcCPUPercent = "metricProcCPUPercent"
	PointTypeMetricProcMemPercent = "metricProcMemPercent"
	PointTypeMetricProcMemRSS     = "metricProcMemRSS"

	PointTypeHost                = "host"
	PointTypeHostBootTime        = "hostBootTime"
	PointKeyHostname             = "hostname"
	PointKeyOS                   = "os"
	PointKeyPlatform             = "platform"
	PointKeyPlatformFamily       = "platformFamily"
	PointKeyPlatformVersion      = "platformVersion"
	PointKeyKernelVersion        = "kernelVersion"
	PointKeyKernelArch           = "kernelArch"
	PointKeyVirtualizationSystem = "virtualizationSystem"
	PointKeyVirtualizationRole   = "virtualizationRole"

	PointKeyUsedPercent = "usedPercent"
	PointKeyTotal       = "total"
	PointKeyAvailable   = "available"
	PointKeyUsed        = "used"
	PointKeyFree        = "free"

	NodeTypeShelly   = "shelly"
	NodeTypeShellyIo = "shellyIo"

	PointTypeSwitch      = "switch"
	PointTypeSwitchSet   = "switchSet"
	PointTypeInput       = "input"
	PointTypeLight       = "light"
	PointTypeLightSet    = "lightSet"
	PointTypeDeviceID    = "deviceID"
	PointTypeIP          = "ip"
	PointTypeVoltage     = "voltage"
	PointTypeCurrent     = "current"
	PointTypePower       = "power"
	PointTypeTemperature = "temp"
	PointTypeBrightness  = "brightness"
	PointTypeWhite       = "white"
	PointTypeLightTemp   = "lightTemp"
	PointTypeTransition  = "transition"
	PointTypeOffline     = "offline"

	PointValueShellyTypeBulbDuo = "BulbDuo"
	PointValueShellyTypeRGBW2   = "rgbw2"
	PointValueShellyType1PM     = "1pm"
	PointValueShellyTypePlugUS  = "PlugUS"
	PointValueShellyTypePlugUK  = "PlugUK"
	PointValueShellyTypePlugIT  = "PlugIT"
	PointValueShellyTypePlugS   = "PlugS"
	PointValueShellyTypeI4      = "PlusI4"
	PointValueShellyTypePlus1   = "Plus1"
	PointValueShellyTypePlus2PM = "Plus2PM"

	PointTypeTimeSync  = "timeSync"
	PointTypeConnected = "connected"

	NodeTypeNetworkManager       = "networkManager"
	NodeTypeNetworkManagerDevice = "networkManagerDevice"
	NodeTypeNetworkManagerConn   = "networkManagerConn"

	NodeTypeNTP             = "ntp"
	PointTypeServer         = "server"
	PointTypeFallbackServer = "fallbackServer"
)
