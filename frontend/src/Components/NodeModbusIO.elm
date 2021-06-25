module Components.NodeModbusIO exposing (view)

import Api.Node exposing (Node)
import Api.Point as Point exposing (Point)
import Color
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Round
import Shared exposing (Msg)
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

        optionInput =
            Form.nodeOptionInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }
                ""
                0

        checkboxInput =
            Form.nodeCheckboxInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }
                ""
                0

        counterWithReset =
            Form.nodeCounterWithReset
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth + 150
                }
                ""
                0

        modbusIOType =
            Point.getText o.node.points "" 0 Point.typeModbusIOType

        isClient =
            case o.parent of
                Just p ->
                    Point.getText p.points "" 0 Point.typeClientServer == Point.valueClient

                Nothing ->
                    False

        isWrite =
            modbusIOType
                == Point.valueModbusHoldingRegister
                || modbusIOType
                == Point.valueModbusCoil

        value : Int -> Float
        value index =
            Point.getValue o.node.points "" index Point.typeValue

        valueSet : Int -> Float
        valueSet index =
            Point.getValue o.node.points "" index Point.typeValueSet

        isRegister =
            modbusIOType
                == Point.valueModbusInputRegister
                || modbusIOType
                == Point.valueModbusHoldingRegister

        isReadOnly =
            Point.getValue o.node.points "" 0 Point.typeReadOnly == 1

        valueText : Int -> String
        valueText index =
            if isRegister then
                String.fromFloat (Round.roundNum 2 (value index))

            else if value index == 0 then
                "off"

            else
                "on"

        twoDigitNumber : Int -> String
        twoDigitNumber num =
            if num > 9 then
                String.fromInt num

            else
                "0" ++ String.fromInt num

        -- Modbus digital input UI element
        di : Int -> String -> Element msg
        di index labelNum =
            row [ spacing 15 ]
                [ Input.text
                    []
                    { onChange =
                        \d ->
                            o.onEditNodePoint (Point "" index Point.typeDescription o.now 0 d 0 0)
                    , text = Point.getText o.node.points "" index Point.typeDescription -- TODO figure out if this works since we already have one description - do we need another type???
                    , placeholder = Just <| Input.placeholder [] <| text "Digital input description"
                    , label = Input.labelLeft [ width (px 150) ] <| el [ alignRight ] <| text <| ("DI_" ++ labelNum ++ ": ")
                    }
                , el [ paddingXY 40 7, Background.color (valueBackgroundColor index), Font.color (valueTextColor index) ] <|
                    text <|
                        valueText index

                -- display the index for reference by user
                , text <|
                    " ("
                        ++ twoDigitNumber index
                        ++ ")"
                ]

        -- Modbus relay UI element
        ry : Int -> String -> Element msg
        ry index labelNum =
            row [ spacing 10 ]
                [ Input.text
                    []
                    { onChange =
                        \d ->
                            o.onEditNodePoint (Point "" index Point.typeDescription o.now 0 d 0 0)
                    , text = Point.getText o.node.points "" index Point.typeDescription -- TODO figure out if this works since we already have one description - do we need another type???
                    , placeholder = Just <| Input.placeholder [] <| text "Relay description"
                    , label = Input.labelLeft [ width (px 150) ] <| el [ alignRight ] <| text <| ("RY_" ++ labelNum ++ ": ")
                    }
                , Form.nodeOnOffInputWithoutColon
                    { onEditNodePoint = o.onEditNodePoint
                    , node = o.node
                    , now = o.now
                    , labelWidth = 0
                    }
                    ""
                    index
                    Point.typeValue
                    Point.typeValueSet
                    ""

                -- display the index for reference by user
                , text <|
                    " ("
                        ++ twoDigitNumber index
                        ++ ")"
                ]

        -- TODO, these should probably take an index as well for consistency
        valueBackgroundColor : Int -> Color
        valueBackgroundColor index =
            if valueText index == "on" then
                Style.colors.blue

            else
                Style.colors.gray

        valueTextColor index =
            if True then
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
            [ Icon.io
            , text <|
                Point.getText o.node.points "" 0 Point.typeDescription
                    ++ ": "

            --, el [ paddingXY 7 0, Background.color valueBackgroundColor, Font.color valueTextColor ] <|
            --   text <|
            --      valueText
            --         ++ (if isRegister then
            --                " " ++ Point.getText o.node.points "" 0 Point.typeUnits
            --
            --                           else
            --                              ""
            --                        )
            --, text <|
            --   if isClient && isWrite && not isReadOnly && value /= valueSet then
            --       " (cmd pending)"
            --
            --               else
            --                  ""
            ]
            :: (if o.expDetail then
                    textInput Point.typeDescription "Description"
                        :: viewIf isClient
                            (numberInput
                                Point.typeID
                                "ID"
                            )
                        :: numberInput Point.typeAddress "Address"
                        :: optionInput Point.typeModbusIOType
                            "IO type"
                            [ ( Point.valueModbusDiscreteInput, "discrete input (r)" )
                            , ( Point.valueModbusCoil, "coil (rw)" )
                            , ( Point.valueModbusInputRegister, "input register (r)" )
                            , ( Point.valueModbusHoldingRegister, "holding register (rw)" )
                            , ( Point.valueModbusWP8024ADAM, "WP8024 - 4 rly, 8 in" )
                            , ( Point.valueModbusWP8025ADAM, "WP8025 - 8 relays" )
                            , ( Point.valueModbusWP8026ADAM, "WP8026 - 16 inputs" )
                            ]
                        :: (viewIf (isClient && isWrite) <|
                                checkboxInput Point.typeReadOnly "Read only"
                           )
                        :: (viewIf isRegister <|
                                numberInput Point.typeScale "Scale factor"
                           )
                        :: (viewIf isRegister <|
                                numberInput Point.typeOffset "Offset"
                           )
                        :: (viewIf isRegister <|
                                textInput Point.typeUnits "Units"
                           )
                        :: (viewIf isRegister <|
                                optionInput Point.typeDataFormat
                                    "Data format"
                                    [ ( Point.valueUINT16, "UINT16" )
                                    , ( Point.valueINT16, "INT16" )
                                    , ( Point.valueUINT32, "UINT32" )
                                    , ( Point.valueINT32, "INT32" )
                                    , ( Point.valueFLOAT32, "FLOAT32" )
                                    ]
                           )
                        -- This can get a little confusing, but client sets the following:
                        --   * coil
                        --   * holding register
                        -- and the server (device) sets the following
                        --   * discrete input
                        --   * input register
                        -- However, some devices also have read only coils and holding regs.
                        -- we can't practically have both the client and server setting a
                        -- value.
                        :: (viewIf
                                (isClient
                                    && modbusIOType
                                    == Point.valueModbusHoldingRegister
                                    && not isReadOnly
                                )
                            <|
                                numberInput
                                    Point.typeValueSet
                                    "Value"
                           )
                        :: (viewIf
                                (isClient
                                    && modbusIOType
                                    == Point.valueModbusCoil
                                    && not isReadOnly
                                )
                            <|
                                onOffInput Point.typeValue Point.typeValueSet "Value"
                           )
                        :: (viewIf (not isClient && modbusIOType == Point.valueModbusInputRegister) <|
                                numberInput Point.typeValue "Value"
                           )
                        :: (viewIf (not isClient && modbusIOType == Point.valueModbusDiscreteInput) <|
                                onOffInput Point.typeValue Point.typeValue "Value"
                           )
                        :: counterWithReset Point.typeErrorCount Point.typeErrorCountReset "Error Count"
                        :: counterWithReset Point.typeErrorCountEOF Point.typeErrorCountEOFReset "EOF Error Count"
                        :: counterWithReset Point.typeErrorCountCRC Point.typeErrorCountCRCReset "CRC Error Count"
                        :: (if modbusIOType == Point.valueModbusWP8024ADAM then
                                [ ry 0 "01"
                                , ry 1 "02"
                                , ry 2 "03"
                                , ry 3 "04"
                                , di 4 "01"
                                , di 5 "02"
                                , di 6 "03"
                                , di 7 "04"
                                , di 8 "05"
                                , di 9 "06"
                                , di 10 "07"
                                , di 11 "08"
                                ]

                            else if modbusIOType == Point.valueModbusWP8025ADAM then
                                [ ry 0 "00"
                                , ry 1 "01"
                                , ry 2 "02"
                                , ry 3 "03"
                                , ry 4 "04"
                                , ry 5 "05"
                                , ry 6 "06"
                                , ry 7 "07"
                                ]

                            else if modbusIOType == Point.valueModbusWP8026ADAM then
                                [ di 0 "00"
                                , di 1 "01"
                                , di 2 "02"
                                , di 3 "03"
                                , di 4 "04"
                                , di 5 "05"
                                , di 6 "06"
                                , di 7 "07"
                                , di 8 "08"
                                , di 9 "09"
                                , di 10 "10"
                                , di 11 "11"
                                , di 12 "12"
                                , di 13 "13"
                                , di 14 "14"
                                , di 15 "15"
                                ]

                            else
                                []
                           )

                else
                    []
               )
