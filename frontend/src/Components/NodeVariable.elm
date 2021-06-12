module Components.NodeVariable exposing (view)

import Api.Node exposing (Node)
import Api.Point as Point exposing (Point)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Round
import Time
import UI.Form as Form
import UI.Icon as Icon
import UI.Style as Style exposing (colors)
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

        value =
            Point.getValue o.node.points "" 0 Point.typeValue

        variableType =
            Point.getText o.node.points "" 0 Point.typeVariableType

        valueText =
            if variableType == Point.valueNumber then
                String.fromFloat (Round.roundNum 2 value)

            else if value == 0 then
                "off"

            else
                "on"

        valueBackgroundColor =
            if valueText == "on" then
                Style.colors.blue

            else
                Style.colors.none

        valueTextColor =
            if valueText == "on" then
                Style.colors.white

            else
                Style.colors.black
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.variable
            , text <|
                Point.getText o.node.points "" 0 Point.typeDescription
            , el [ paddingXY 7 0, Background.color valueBackgroundColor, Font.color valueTextColor ] <|
                text <|
                    valueText
                        ++ (if variableType == Point.valueNumber then
                                " " ++ Point.getText o.node.points "" 0 Point.typeUnits

                            else
                                ""
                           )
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeDescription "Description"
                    , optionInput Point.typeVariableType
                        "Variable type"
                        [ ( Point.valueOnOff, "On/Off" )
                        , ( Point.valueNumber, "Number" )
                        ]
                    , viewIf (variableType == Point.valueOnOff) <|
                        onOffInput
                            Point.typeValue
                            Point.typeValue
                            "Value"
                    , viewIf (variableType == Point.valueNumber) <|
                        numberInput Point.typeValue "Value"
                    , viewIf (variableType == Point.valueNumber) <|
                        textInput Point.typeUnits "Units"
                    ]

                else
                    []
               )
