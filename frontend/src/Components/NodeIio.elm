module Components.NodeIio exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
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


view : NodeOptions msg -> Element msg
view o =
    let
        disabled =
            Point.getBool o.node.points Point.typeDisabled ""

        connected =
            Point.getBool o.node.points Point.typeConnected ""

        -- the driver's own name for the device, which is what confirms the
        -- node found what it was pointed at
        deviceName =
            Point.getText o.node.points Point.typeDeviceName ""
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.io
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            , viewIf (deviceName /= "") <|
                el [ paddingXY 7 0 ] <|
                    text deviceName
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

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        counterWithReset =
                            NodeInputs.nodeCounterWithReset opts "0"

                        status label v =
                            text <| "  " ++ label ++ ": " ++ v

                        error =
                            Point.getText o.node.points Point.typeError ""
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typeDevice "Device" "ads1015"
                    , numberInput Point.typePollPeriod "Poll period (ms)"
                    , numberInput Point.typeSampleFrequency "Sample frequency (Hz)"
                    , numberInput Point.typeOversampling "Oversampling ratio"
                    , checkboxInput Point.typeDisabled "Disabled"
                    , numberInput Point.typeDebug "Debug level (0-9)"
                    , horizontalRule
                    , status "Connected" <|
                        if connected then
                            "yes"

                        else
                            "no"
                    , status "Device name" deviceName
                    , status "Device path" <|
                        Point.getText o.node.points Point.typeDevicePath ""
                    , viewIf (error /= "") <| status "Error" error
                    , counterWithReset Point.typeErrorCount
                        Point.typeErrorCountReset
                        "Error count"
                    , NodeInputs.nodeKeyValueInput opts Point.typeTag "Tags" "Add Tag"
                    ]

                else
                    []
               )
