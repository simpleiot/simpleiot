module Components.NodeCondition exposing (view)

import Api.Node exposing (Node)
import Api.Point as Point exposing (Point)
import Element exposing (..)
import Element.Border as Border
import Time
import UI.Form as Form
import UI.Icon as Icon
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view :
    { isRoot : Bool
    , now : Time.Posix
    , zone : Time.Zone
    , modified : Bool
    , expDetail : Bool
    , parent : Maybe Node
    , node : Node
    , onApiDelete : String -> msg
    , onEditNodePoint : String -> Point -> msg
    , onDiscardEdits : msg
    , onApiPostPoints : String -> msg
    , onClipboard : String -> msg
    }
    -> Element msg
view o =
    let
        labelWidth =
            150

        textInput =
            Form.nodeTextInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }

        numberInput =
            Form.nodeNumberInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }

        optionInput =
            Form.nodeOptionInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }

        onOffInput =
            Form.nodeOnOffInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }

        conditionType =
            Point.getText o.node.points Point.typeConditionType

        operators =
            case conditionType of
                "value" ->
                    [ ( Point.valueGreaterThan, ">" )
                    , ( Point.valueLessThan, "<" )
                    , ( Point.valueEqual, "=" )
                    , ( Point.valueNotEqual, "!=" )
                    ]

                "valueText" ->
                    [ ( Point.valueEqual, "=" )
                    , ( Point.valueNotEqual, "!=" )
                    ]

                "sysState" ->
                    [ ( Point.valueEqual, "=" )
                    , ( Point.valueNotEqual, "!=" )
                    ]

                _ ->
                    []
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.check
            , text <|
                Point.getText o.node.points Point.typeDescription
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeDescription "Description"
                    , textInput Point.typeID "Node ID"
                    , optionInput Point.typeConditionType
                        "Attribute"
                        [ ( Point.valueConditionValue, "value" )
                        , ( Point.valueConditionValueBool, "on/off" )
                        , ( Point.valueConditionValueText, "text" )
                        , ( Point.valueConditionSystemState, "system state" )
                        ]
                    , if conditionType /= Point.valueConditionValueBool then
                        optionInput Point.typeOperator "Operator" operators

                      else
                        Element.none
                    , case conditionType of
                        "value" ->
                            numberInput Point.typeValue "Value"

                        "valueBool" ->
                            onOffInput Point.typeValue Point.typeValue "Value"

                        "valueText" ->
                            textInput Point.typeValue "Value"

                        "sysState" ->
                            optionInput Point.typeValue
                                "Value"
                                [ ( Point.valueSysStatePowerOff, "power off" )
                                , ( Point.valueSysStateOffline, "offline" )
                                , ( Point.valueSysStateOnline, "online" )
                                ]

                        _ ->
                            Element.none
                    , viewIf o.modified <|
                        Form.buttonRow
                            [ Form.button
                                { label = "save"
                                , color = colors.blue
                                , onPress = o.onApiPostPoints o.node.id
                                }
                            , Form.button
                                { label = "discard"
                                , color = colors.gray
                                , onPress = o.onDiscardEdits
                                }
                            ]
                    ]

                else
                    []
               )
