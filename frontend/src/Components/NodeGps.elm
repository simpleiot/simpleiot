module Components.NodeGps exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Round
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style
import UI.ViewIf exposing (viewIf)


horizontalRule : Element msg
horizontalRule =
    el
        [ width fill
        , height (px 1)
        , Border.color (Element.rgb 0.5 0.5 0.5)
        , Border.widthEach { bottom = 2, top = 0, left = 0, right = 0 }
        ]
        Element.none


{-| Fix type is stored as a number so it can be graphed and stored in
metrics-only databases. The encoding follows gpsd's TPV mode field.
-}
fixTypeLabel : Float -> String
fixTypeLabel v =
    case round v of
        2 ->
            "2D"

        3 ->
            "3D"

        _ ->
            "no fix"


{-| Fix quality is likewise numeric, following the NMEA GGA fix quality
encoding, which covers everything the three sources report.
-}
fixQualityLabel : Float -> String
fixQualityLabel v =
    case round v of
        1 ->
            "GPS"

        2 ->
            "DGPS"

        3 ->
            "PPS"

        4 ->
            "RTK fixed"

        5 ->
            "RTK float"

        6 ->
            "estimated"

        7 ->
            "manual"

        8 ->
            "simulated"

        _ ->
            "none"


{-| Coordinates are shown to five decimal places, which is roughly a meter.
-}
coordinate : Float -> String
coordinate v =
    Round.round 5 v


view : NodeOptions msg -> Element msg
view o =
    let
        disabled =
            Point.getBool o.node.points Point.typeDisabled ""

        connected =
            Point.getBool o.node.points Point.typeConnected ""

        source =
            Point.getText o.node.points Point.typeGpsSource ""

        latitude =
            Point.getValue o.node.points Point.typeLatitude ""

        longitude =
            Point.getValue o.node.points Point.typeLongitude ""

        summaryBackground =
            if disabled || not connected then
                Style.colors.ltgray

            else
                Style.colors.none
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10, Background.color summaryBackground ]
            [ Icon.mapPin
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            , el [ paddingXY 7 0 ] <|
                text <|
                    coordinate latitude
                        ++ ", "
                        ++ coordinate longitude
            , viewIf disabled <| text "(disabled)"
            , viewIf (not disabled && not connected) <| text "(not connected)"
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            180

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        numberInput =
                            NodeInputs.nodeNumberInput opts "0"

                        optionInput =
                            NodeInputs.nodeOptionInput opts "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        counterWithReset =
                            NodeInputs.nodeCounterWithReset opts "0"

                        -- an unset source behaves as serial, matching the
                        -- backend default
                        isSerial =
                            source == Point.valueGpsSourceSerial || source == ""

                        isGpsd =
                            source == Point.valueGpsSourceGpsd

                        isSim =
                            source == Point.valueGpsSourceSim

                        status label value =
                            text <| "  " ++ label ++ ": " ++ value

                        rounded places typ =
                            Round.round places <|
                                Point.getValue o.node.points typ ""
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , checkboxInput Point.typeDisabled "Disabled"
                    , optionInput Point.typeGpsSource
                        "Source"
                        [ ( Point.valueGpsSourceSerial, "Serial (NMEA)" )
                        , ( Point.valueGpsSourceGpsd, "gpsd" )
                        , ( Point.valueGpsSourceSim, "Simulated" )
                        ]
                    , viewIf isSerial <|
                        textInput Point.typePort "Port" "/dev/ttyUSB0"
                    , viewIf isSerial <|
                        textInput Point.typeBaud "Baud" "9600"
                    , viewIf isGpsd <|
                        textInput Point.typeGpsdAddress
                            "gpsd address"
                            "localhost:2947"
                    , viewIf isGpsd <|
                        textInput Point.typeDevice
                            "Device"
                            "(blank watches all)"
                    , viewIf isSim <|
                        numberInput Point.typeSimLatitude "Start latitude"
                    , viewIf isSim <|
                        numberInput Point.typeSimLongitude "Start longitude"
                    , viewIf isSim <|
                        numberInput Point.typeSimSpeed "Speed (m/s)"
                    , viewIf isSim <|
                        numberInput Point.typeSimHeading "Start heading (deg)"
                    , viewIf isSim <|
                        numberInput Point.typeSimHeadingRate
                            "Heading drift (deg/s)"
                    , viewIf isSim <|
                        numberInput Point.typePeriod "Update period (s)"
                    , numberInput Point.typeDebug "Debug level (0-9)"
                    , horizontalRule
                    , status "Latitude" <| coordinate latitude
                    , status "Longitude" <| coordinate longitude
                    , status "Altitude (m)" <| rounded 1 Point.typeAltitude
                    , status "Speed (m/s)" <| rounded 2 Point.typeSpeed
                    , status "Heading (deg)" <| rounded 1 Point.typeHeading
                    , status "Fix type" <|
                        fixTypeLabel <|
                            Point.getValue o.node.points Point.typeFixType ""
                    , status "Fix quality" <|
                        fixQualityLabel <|
                            Point.getValue o.node.points Point.typeFixQuality ""
                    , status "Satellites" <| rounded 0 Point.typeNumSat
                    , status "HDOP" <| rounded 2 Point.typeHdop
                    , status "Connected" <|
                        if connected then
                            "yes"

                        else
                            "no"
                    , counterWithReset Point.typeRx Point.typeRxReset "Rx count"
                    , counterWithReset Point.typeErrorCount
                        Point.typeErrorCountReset
                        "Error count"
                    , NodeInputs.nodeKeyValueInput opts Point.typeTag "Tags" "Add Tag"
                    ]

                else
                    []
               )
