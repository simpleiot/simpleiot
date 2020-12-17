module Components.NodeModbusIO exposing (view)

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

        onOffInput =
            Form.nodeOnOffInput
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

        modbusIOType =
            Point.getText o.node.points Point.typeModbusIOType

        isClient =
            case o.parent of
                Just p ->
                    Point.getText p.points Point.typeClientServer == Point.valueClient

                Nothing ->
                    False

        value =
            Point.getValue o.node.points Point.typeValue

        valueSet =
            Point.getValue o.node.points Point.typeValueSet

        isRegister =
            modbusIOType
                == Point.valueModbusInputRegister
                || modbusIOType
                == Point.valueModbusHoldingRegister

        valueText =
            if isRegister then
                String.fromFloat value

            else if value == 0 then
                "off"

            else
                "on"
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
                Point.getText o.node.points Point.typeDescription
                    ++ ": "
                    ++ valueText
                    ++ (if isRegister then
                            " " ++ Point.getText o.node.points Point.typeUnits

                        else
                            ""
                       )
                    ++ (if value /= valueSet then
                            " (cmd pending)"

                        else
                            ""
                       )
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

                    -- this can get a little confusing, but client sets the following:
                    --   * coil
                    --   * holding register
                    -- and the server sets the following
                    --   * discrete input
                    --   * input register
                    -- we can't practically have both the client and server setting a
                    -- value.
                    , viewIf (isClient && modbusIOType == Point.valueModbusHoldingRegister) <|
                        numberInput Point.typeValueSet "Value"
                    , viewIf (isClient && modbusIOType == Point.valueModbusCoil) <|
                        onOffInput Point.typeValue Point.typeValueSet "Value"
                    , viewIf (not isClient && modbusIOType == Point.valueModbusInputRegister) <|
                        numberInput Point.typeValue "Value"
                    , viewIf (not isClient && modbusIOType == Point.valueModbusDiscreteInput) <|
                        onOffInput Point.typeValue Point.typeValue "Value"
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
