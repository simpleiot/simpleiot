module Components.NodeCondition exposing (view)

import Api.Node exposing (Node)
import Api.Point as Point exposing (Point)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Time
import UI.Form as Form
import UI.Icon as Icon
import UI.Style as Style exposing (colors)


view :
    { now : Time.Posix
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

        conditionType =
            Point.getText o.node.points "" 0 Point.typeConditionType

        active =
            Point.getBool o.node.points "" 0 Point.typeActive

        descBackgroundColor =
            if active then
                Style.colors.blue

            else
                Style.colors.none

        descTextColor =
            if active then
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
            [ Icon.check
            , el [ Background.color descBackgroundColor, Font.color descTextColor ] <|
                text <|
                    Point.getText o.node.points "" 0 Point.typeDescription
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeDescription "Description"
                    , optionInput Point.typeConditionType
                        "Type"
                        [ ( Point.valuePointValue, "point value" )
                        , ( Point.valueSchedule, "schedule" )
                        ]
                    , case conditionType of
                        "pointValue" ->
                            pointValue o labelWidth

                        "schedule" ->
                            schedule o labelWidth

                        _ ->
                            text "Please select condition type"
                    ]

                else
                    []
               )


schedule :
    { now : Time.Posix
    , zone : Time.Zone
    , modified : Bool
    , expDetail : Bool
    , parent : Maybe Node
    , node : Node
    , onEditNodePoint : Point -> msg
    }
    -> Int
    -> Element msg
schedule o labelWidth =
    let
        timeInput =
            Form.nodeTimeInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , zone = o.zone
                , labelWidth = labelWidth
                }
                ""
                0

        weekdayCheckboxInput index label =
            column []
                [ text label
                , Form.nodeCheckboxInput
                    { onEditNodePoint = o.onEditNodePoint
                    , node = o.node
                    , now = o.now
                    , labelWidth = 0
                    }
                    ""
                    index
                    Point.typeWeekday
                    ""
                ]
    in
    column
        [ width fill
        , spacing 6
        , paddingEach { top = 15, right = 0, bottom = 0, left = 0 }
        ]
        [ wrappedRow [ spacing 10, paddingEach { top = 0, right = 0, bottom = 5, left = labelWidth } ]
            -- here, number matches Go Weekday definitions
            -- https://pkg.go.dev/time#Weekday
            [ weekdayCheckboxInput 0 " S"
            , weekdayCheckboxInput 1 " M"
            , weekdayCheckboxInput 2 " T"
            , weekdayCheckboxInput 3 " W"
            , weekdayCheckboxInput 4 " T"
            , weekdayCheckboxInput 5 " F"
            , weekdayCheckboxInput 6 " S"
            ]
        , timeInput Point.typeStart "Start time"
        , timeInput Point.typeEnd "End time"
        ]


pointValue :
    { now : Time.Posix
    , zone : Time.Zone
    , modified : Bool
    , expDetail : Bool
    , parent : Maybe Node
    , node : Node
    , onEditNodePoint : Point -> msg
    }
    -> Int
    -> Element msg
pointValue o labelWidth =
    let
        textInput =
            Form.nodeTextInput
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

        optionInput =
            Form.nodeOptionInput
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

        conditionValueType =
            Point.getText o.node.points "" 0 Point.typeValueType

        operators =
            case conditionValueType of
                "number" ->
                    [ ( Point.valueGreaterThan, ">" )
                    , ( Point.valueLessThan, "<" )
                    , ( Point.valueEqual, "=" )
                    , ( Point.valueNotEqual, "!=" )
                    ]

                "text" ->
                    [ ( Point.valueEqual, "=" )
                    , ( Point.valueNotEqual, "!=" )
                    , ( Point.valueContains, "contains" )
                    ]

                _ ->
                    []
    in
    column
        [ width fill
        , spacing 6
        ]
        [ textInput Point.typeID "Node ID"
        , optionInput Point.typePointType
            "Point Type"
            [ ( Point.typeValue, "value" )
            , ( Point.typeValueSet, "set value" )
            , ( Point.typeErrorCount, "error count" )
            , ( Point.typeSysState, "system state" )
            ]
        , textInput Point.typePointID "Point ID"
        , numberInput Point.typePointIndex "Point Index"
        , optionInput Point.typeValueType
            "Point Value Type"
            [ ( Point.valueNumber, "number" )
            , ( Point.valueOnOff, "on/off" )
            , ( Point.valueText, "text" )
            ]
        , if conditionValueType /= Point.valueOnOff then
            optionInput Point.typeOperator "Operator" operators

          else
            Element.none
        , case conditionValueType of
            "number" ->
                numberInput Point.typeValue "Point Value"

            "onOff" ->
                onOffInput Point.typeValue Point.typeValue "Point Value"

            "text" ->
                textInput Point.typeValue "Point Value"

            _ ->
                Element.none
        , numberInput Point.typeMinActive "Min active time (m)"
        ]
