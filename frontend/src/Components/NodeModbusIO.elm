module Components.NodeModbusIO exposing (view)

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

        value =
            Point.getValue o.node.points "" 0 Point.typeValue

        valueSet =
            Point.getValue o.node.points "" 0 Point.typeValueSet

        isRegister =
            modbusIOType
                == Point.valueModbusInputRegister
                || modbusIOType
                == Point.valueModbusHoldingRegister

        isReadOnly =
            Point.getValue o.node.points "" 0 Point.typeReadOnly == 1

        valueText =
            if isRegister then
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
            [ Icon.io
            , text <|
                Point.getText o.node.points "" 0 Point.typeDescription
                    ++ ": "
            , el [ paddingXY 7 0, Background.color valueBackgroundColor, Font.color valueTextColor ] <|
                text <|
                    valueText
                        ++ (if isRegister then
                                " " ++ Point.getText o.node.points "" 0 Point.typeUnits

                            else
                                ""
                           )
            , text <|
                if isClient && isWrite && not isReadOnly && value /= valueSet then
                    " (cmd pending)"

                else
                    ""
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeDescription "Description"
                    , viewIf isClient <| numberInput Point.typeID "ID"
                    , numberInput Point.typeAddress "Address"
                    , optionInput Point.typeModbusIOType
                        "IO type"
                        [ ( Point.valueModbusDiscreteInput, "discrete input (r)" )
                        , ( Point.valueModbusCoil, "coil (rw)" )
                        , ( Point.valueModbusInputRegister, "input register(r)" )
                        , ( Point.valueModbusHoldingRegister, "holding register(rw)" )
                        ]
                    , viewIf (isClient && isWrite) <|
                        checkboxInput Point.typeReadOnly "Read only"
                    , viewIf isRegister <|
                        numberInput Point.typeScale "Scale factor"
                    , viewIf isRegister <|
                        numberInput Point.typeOffset "Offset"
                    , viewIf isRegister <|
                        textInput Point.typeUnits "Units"
                    , viewIf isRegister <|
                        optionInput Point.typeDataFormat
                            "Data format"
                            [ ( Point.valueUINT16, "UINT16" )
                            , ( Point.valueINT16, "INT16" )
                            , ( Point.valueUINT32, "UINT32" )
                            , ( Point.valueINT32, "INT32" )
                            , ( Point.valueFLOAT32, "FLOAT32" )
                            ]

                    -- This can get a little confusing, but client sets the following:
                    --   * coil
                    --   * holding register
                    -- and the server (device) sets the following
                    --   * discrete input
                    --   * input register
                    -- However, some devices also have read only coils and holding regs.
                    -- we can't practically have both the client and server setting a
                    -- value.
                    , viewIf
                        (isClient
                            && modbusIOType
                            == Point.valueModbusHoldingRegister
                            && not isReadOnly
                        )
                      <|
                        numberInput Point.typeValueSet "Value"
                    , viewIf
                        (isClient
                            && modbusIOType
                            == Point.valueModbusCoil
                            && not isReadOnly
                        )
                      <|
                        onOffInput Point.typeValue Point.typeValueSet "Value"
                    , viewIf (not isClient && modbusIOType == Point.valueModbusInputRegister) <|
                        numberInput Point.typeValue "Value"
                    , viewIf (not isClient && modbusIOType == Point.valueModbusDiscreteInput) <|
                        onOffInput Point.typeValue Point.typeValue "Value"
                    , counterWithReset Point.typeErrorCount Point.typeErrorCountReset "Error Count"
                    , counterWithReset Point.typeErrorCountEOF Point.typeErrorCountEOFReset "EOF Error Count"
                    , counterWithReset Point.typeErrorCountCRC Point.typeErrorCountCRCReset "CRC Error Count"
                    ]

                else
                    []
               )
