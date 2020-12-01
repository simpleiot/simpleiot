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
    , node : Node
    , onApiDelete : String -> msg
    , onEditNodePoint : String -> Point -> msg
    , onDiscardEdits : msg
    , onApiPostPoints : String -> msg
    }
    -> Element msg
view o =
    let
        textInput2 =
            Form.nodeTextInput { onEditNodePoint = o.onEditNodePoint, node = o.node, now = o.now }

        modbusIOType =
            Point.getPointText o.node.points Point.typeModbusIOType
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
                Point.getPointText o.node.points Point.typeFirstName
                    ++ " "
                    ++ Point.getPointText o.node.points Point.typeLastName
            , viewIf o.isRoot <|
                Icon.x (o.onApiDelete o.node.id)
            ]
            :: (if o.expDetail then
                    [ textInput2 Point.typeDescription "Description"
                    , textInput2 Point.typeID "ID"
                    , textInput2 Point.typeAddress "Address"
                    , textInput2 Point.typeModbusIOType "IO Type"
                    , viewIf (modbusIOType == Point.valueModbusRegister) <|
                        textInput2 Point.typeScale "Scale factor"
                    , viewIf (modbusIOType == Point.valueModbusRegister) <|
                        textInput2 Point.typeOffset "Offset"
                    , viewIf (modbusIOType == Point.valueModbusRegister) <|
                        textInput2 Point.typeUnits "Units"
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
