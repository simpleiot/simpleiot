module Components.NodeAction exposing (view)

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
    , onEditNodePoint : Point -> msg
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
                ""
                0

        optionInput =
            Form.nodeOptionInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }
                ""
                0

        numberInput =
            Form.nodeNumberInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }
                ""
                0

        onOffInput =
            Form.nodeOnOffInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }
                ""
                0

        actionType =
            Point.getText o.node.points "" 0 Point.typeActionType

        actionSetValue =
            actionType == Point.valueActionSetValue

        valueType =
            Point.getText o.node.points "" 0 Point.typeValueType
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.trendingUp
            , text <|
                Point.getText o.node.points "" 0 Point.typeDescription
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeDescription "Description"
                    , optionInput Point.typeActionType
                        "Action"
                        [ ( Point.valueActionNotify, "notify" )
                        , ( Point.valueActionSetValue, "set node value" )
                        ]
                    , viewIf actionSetValue <|
                        optionInput Point.typePointType
                            "Point Type"
                            [ ( Point.typeValue, "value" )
                            , ( Point.typeValueSet, "set value (use for remote devices)" )
                            ]
                    , viewIf actionSetValue <| textInput Point.typeID "Node ID"
                    , viewIf actionSetValue <|
                        optionInput Point.typeValueType
                            "Point Value Type"
                            [ ( Point.valueNumber, "number" )
                            , ( Point.valueOnOff, "on/off" )
                            , ( Point.valueText, "text" )
                            ]
                    , viewIf actionSetValue <|
                        case valueType of
                            "number" ->
                                numberInput Point.typeValue "Value"

                            "onOff" ->
                                onOffInput Point.typeValue Point.typeValue "Value"

                            "text" ->
                                textInput Point.typeValue "Value"

                            _ ->
                                Element.none
                    ]

                else
                    []
               )
