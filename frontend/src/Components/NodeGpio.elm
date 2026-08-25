module Components.NodeGpio exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
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

        value =
            Point.getBool o.node.points Point.typeValue ""

        -- a simulated line has no hardware behind it, which is worth saying
        -- where someone can see it without opening the node
        isSim =
            Point.getText o.node.points Point.typeChip "" == Point.valueSim

        valueText =
            if value then
                "on"

            else
                "off"

        valueBackgroundColor =
            if value then
                colors.blue

            else
                colors.none

        valueTextColor =
            if value then
                colors.white

            else
                colors.black
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
                    ++ ": "
            , el
                [ paddingXY 7 0
                , Background.color valueBackgroundColor
                , Font.color valueTextColor
                ]
              <|
                text valueText
            , viewIf isSim <| text "(simulated)"
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

                        onOffInput =
                            NodeInputs.nodeOnOffInput opts "0"

                        optionInput =
                            NodeInputs.nodeOptionInput opts "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        counterWithReset =
                            NodeInputs.nodeCounterWithReset opts "0"

                        status label v =
                            text <| "  " ++ label ++ ": " ++ v

                        -- an unset direction is an input, matching the
                        -- backend default
                        isOutput =
                            Point.getText o.node.points Point.typeDirection ""
                                == Point.valueOutput

                        error =
                            Point.getText o.node.points Point.typeError ""
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typeChip "Chip" "gpiochip0"
                    , textInput Point.typeLine "Line" "offset or name"
                    , optionInput Point.typeDirection
                        "Direction"
                        [ ( Point.valueInput, "Input" )
                        , ( Point.valueOutput, "Output" )
                        ]
                    , viewIf (not isOutput) <|
                        optionInput Point.typeBias
                            "Bias"
                            [ ( "", "As is" )
                            , ( Point.valuePullUp, "Pull up" )
                            , ( Point.valuePullDown, "Pull down" )
                            , ( Point.valueBiasDisabled, "Disabled" )
                            ]
                    , viewIf isOutput <|
                        optionInput Point.typeDrive
                            "Drive"
                            [ ( Point.valuePushPull, "Push-pull" )
                            , ( Point.valueOpenDrain, "Open drain" )
                            , ( Point.valueOpenSource, "Open source" )
                            ]
                    , checkboxInput Point.typeActiveLow "Active low"
                    , viewIf (not isOutput) <|
                        numberInput Point.typeDebounce "Debounce (ms)"
                    , viewIf (not isOutput) <|
                        numberInput Point.typePollPeriod "Poll period (ms)"
                    , viewIf isOutput <|
                        checkboxInput Point.typeInitialValue "Initial value"
                    , viewIf isOutput <|
                        onOffInput Point.typeValue Point.typeValueSet "Value"
                    , checkboxInput Point.typeDisabled "Disabled"
                    , numberInput Point.typeDebug "Debug level (0-9)"
                    , horizontalRule
                    , status "Value" valueText
                    , status "Connected" <|
                        if connected then
                            "yes"

                        else
                            "no"
                    , status "Line offset" <|
                        String.fromInt <|
                            round <|
                                Point.getValue o.node.points Point.typeLineOffset ""
                    , status "Line name" <|
                        Point.getText o.node.points Point.typeLineName ""
                    , viewIf (error /= "") <| status "Error" error
                    , counterWithReset Point.typeErrorCount
                        Point.typeErrorCountReset
                        "Error count"
                    , NodeInputs.nodeKeyValueInput opts Point.typeTag "Tags" "Add Tag"
                    ]

                else
                    []
               )
